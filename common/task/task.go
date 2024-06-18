package task

import (
	"context"
	"errors"
	"sync"

	"github.com/lumavpn/luma/util"
)

type taskItem struct {
	Name string
	Run  func(ctx context.Context) error
}

type errTaskSucceed struct{}

func (e errTaskSucceed) Error() string {
	return "task succeed"
}

type Group struct {
	tasks    []taskItem
	cleanup  func()
	fastFail bool
	queue    chan struct{}
}

func (g *Group) Append(name string, f func(ctx context.Context) error) {
	g.tasks = append(g.tasks, taskItem{
		Name: name,
		Run:  f,
	})
}

func (g *Group) Append0(f func(ctx context.Context) error) {
	g.tasks = append(g.tasks, taskItem{
		Run: f,
	})
}

func (g *Group) Cleanup(f func()) {
	g.cleanup = f
}

func (g *Group) FastFail() {
	g.fastFail = true
}

func (g *Group) Concurrency(n int) {
	g.queue = make(chan struct{}, n)
	for i := 0; i < n; i++ {
		g.queue <- struct{}{}
	}
}

func (g *Group) Run(contextList ...context.Context) error {
	return g.RunContextList(contextList)
}

func (g *Group) RunContextList(contextList []context.Context) error {
	if len(contextList) == 0 {
		contextList = append(contextList, context.Background())
	}

	taskContext, taskFinish := context.WithCancelCause(context.Background())
	taskCancelContext, taskCancel := context.WithCancelCause(context.Background())

	var errorAccess sync.Mutex
	var returnError error
	taskCount := len(g.tasks)

	for _, task := range g.tasks {
		currentTask := task
		go func() {
			if g.queue != nil {
				select {
				case <-taskCancelContext.Done():
					errorAccess.Lock()
					taskCount--
					currentCount := taskCount
					if currentCount == 0 {
						taskCancel(errTaskSucceed{})
						taskFinish(errTaskSucceed{})
					}
					errorAccess.Unlock()
					return
				case <-g.queue:
				}
			}
			err := currentTask.Run(taskCancelContext)
			errorAccess.Lock()
			if err != nil {
				if currentTask.Name != "" {
					err = errors.New(currentTask.Name)
				}
				returnError = err
				if g.fastFail {
					taskCancel(err)
				}
			}
			taskCount--
			currentCount := taskCount
			errorAccess.Unlock()
			if currentCount == 0 {
				taskCancel(errTaskSucceed{})
				taskFinish(errTaskSucceed{})
			}
			if g.queue != nil {
				g.queue <- struct{}{}
			}
		}()
	}

	selectedContext, upstreamErr := util.SelectContext(append([]context.Context{taskCancelContext}, contextList...))

	if selectedContext != 0 {
		taskCancel(upstreamErr)
	}

	if g.cleanup != nil {
		g.cleanup()
	}

	<-taskContext.Done()

	return returnError
}
