package mmdb

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	C "github.com/lumavpn/luma/common"
	lumaOnce "github.com/lumavpn/luma/common/once"
	lumaHttp "github.com/lumavpn/luma/component/http"
	"github.com/lumavpn/luma/log"

	"github.com/oschwald/maxminddb-golang"
)

type databaseType = uint8

const (
	typeMaxmind databaseType = iota
	typeSing
	typeMetaV0
)

var (
	IPreader  IPReader
	ASNreader ASNReader
	IPonce    sync.Once
	ASNonce   sync.Once
)

func LoadFromBytes(buffer []byte) {
	IPonce.Do(func() {
		mmdb, err := maxminddb.FromBytes(buffer)
		if err != nil {
			log.Fatalf("Can't load mmdb: %s", err.Error())
		}
		IPreader = IPReader{Reader: mmdb}
		switch mmdb.Metadata.DatabaseType {
		case "sing-geoip":
			IPreader.databaseType = typeSing
		case "Meta-geoip0":
			IPreader.databaseType = typeMetaV0
		default:
			IPreader.databaseType = typeMaxmind
		}
	})
}

func Verify(path string) bool {
	instance, err := maxminddb.Open(path)
	if err == nil {
		instance.Close()
	}
	return err == nil
}

func IPInstance() IPReader {
	IPonce.Do(func() {
		mmdbPath := C.Path.MMDB()
		log.Infof("Load MMDB file: %s", mmdbPath)
		mmdb, err := maxminddb.Open(mmdbPath)
		if err != nil {
			log.Fatalf("Can't load MMDB: %s", err.Error())
		}
		IPreader = IPReader{Reader: mmdb}
		switch mmdb.Metadata.DatabaseType {
		case "sing-geoip":
			IPreader.databaseType = typeSing
		case "Meta-geoip0":
			IPreader.databaseType = typeMetaV0
		default:
			IPreader.databaseType = typeMaxmind
		}
	})

	return IPreader
}

func DownloadMMDB(path string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
	defer cancel()
	resp, err := lumaHttp.HttpRequest(ctx, C.MmdbUrl, http.MethodGet, http.Header{"User-Agent": {C.UA}}, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)

	return err
}

func ASNInstance() ASNReader {
	ASNonce.Do(func() {
		ASNPath := C.Path.ASN()
		log.Infof("Load ASN file: %s", ASNPath)
		asn, err := maxminddb.Open(ASNPath)
		if err != nil {
			log.Fatalf("Can't load ASN: %s", err.Error())
		}
		ASNreader = ASNReader{Reader: asn}
	})

	return ASNreader
}

func DownloadASN(path string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
	defer cancel()
	resp, err := lumaHttp.HttpRequest(ctx, C.ASNUrl, http.MethodGet, http.Header{"User-Agent": {C.UA}}, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)

	return err
}

func ReloadIP() {
	lumaOnce.Reset(&IPonce)
}

func ReloadASN() {
	lumaOnce.Reset(&ASNonce)
}
