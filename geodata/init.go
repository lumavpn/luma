package geodata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	C "github.com/lumavpn/luma/common"
	lumaHttp "github.com/lumavpn/luma/component/http"
	"github.com/lumavpn/luma/component/mmdb"
	"github.com/lumavpn/luma/log"
)

var (
	initGeoSite bool
	initGeoIP   int
	initASN     bool
)

func InitGeoSite() error {
	if _, err := os.Stat(C.Path.GeoSite()); os.IsNotExist(err) {
		log.Infof("Can't find GeoSite.dat, start download")
		if err := downloadGeoSite(C.Path.GeoSite()); err != nil {
			return fmt.Errorf("can't download GeoSite.dat: %s", err.Error())
		}
		log.Infof("Download GeoSite.dat finish")
		initGeoSite = false
	}
	if !initGeoSite {
		if err := Verify(C.GeositeName); err != nil {
			log.Warnf("GeoSite.dat invalid, remove and download: %s", err)
			if err := os.Remove(C.Path.GeoSite()); err != nil {
				return fmt.Errorf("can't remove invalid GeoSite.dat: %s", err.Error())
			}
			if err := downloadGeoSite(C.Path.GeoSite()); err != nil {
				return fmt.Errorf("can't download GeoSite.dat: %s", err.Error())
			}
		}
		initGeoSite = true
	}
	return nil
}

func downloadGeoSite(path string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
	defer cancel()
	resp, err := lumaHttp.HttpRequest(ctx, C.GeoSiteUrl, http.MethodGet, http.Header{"User-Agent": {C.UA}}, nil)
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

func downloadGeoIP(path string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
	defer cancel()
	resp, err := lumaHttp.HttpRequest(ctx, C.GeoIpUrl, http.MethodGet, http.Header{"User-Agent": {C.UA}}, nil)
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

func InitGeoIP() error {
	if C.GeodataMode {
		if _, err := os.Stat(C.Path.GeoIP()); os.IsNotExist(err) {
			log.Infof("Can't find GeoIP.dat, start download")
			if err := downloadGeoIP(C.Path.GeoIP()); err != nil {
				return fmt.Errorf("can't download GeoIP.dat: %s", err.Error())
			}
			log.Infof("Download GeoIP.dat finish")
			initGeoIP = 0
		}

		if initGeoIP != 1 {
			if err := Verify(C.GeoipName); err != nil {
				log.Warnf("GeoIP.dat invalid, remove and download: %s", err)
				if err := os.Remove(C.Path.GeoIP()); err != nil {
					return fmt.Errorf("can't remove invalid GeoIP.dat: %s", err.Error())
				}
				if err := downloadGeoIP(C.Path.GeoIP()); err != nil {
					return fmt.Errorf("can't download GeoIP.dat: %s", err.Error())
				}
			}
			initGeoIP = 1
		}
		return nil
	}

	if _, err := os.Stat(C.Path.MMDB()); os.IsNotExist(err) {
		log.Infof("Can't find MMDB, start download")
		if err := mmdb.DownloadMMDB(C.Path.MMDB()); err != nil {
			return fmt.Errorf("can't download MMDB: %s", err.Error())
		}
	}

	if initGeoIP != 2 {
		if !mmdb.Verify(C.Path.MMDB()) {
			log.Warnf("MMDB invalid, remove and download")
			if err := os.Remove(C.Path.MMDB()); err != nil {
				return fmt.Errorf("can't remove invalid MMDB: %s", err.Error())
			}
			if err := mmdb.DownloadMMDB(C.Path.MMDB()); err != nil {
				return fmt.Errorf("can't download MMDB: %s", err.Error())
			}
		}
		initGeoIP = 2
	}
	return nil
}

func InitASN() error {
	if _, err := os.Stat(C.Path.ASN()); os.IsNotExist(err) {
		log.Infof("Can't find ASN.mmdb, start download")
		if err := mmdb.DownloadASN(C.Path.ASN()); err != nil {
			return fmt.Errorf("can't download ASN.mmdb: %s", err.Error())
		}
		log.Infof("Download ASN.mmdb finish")
		initASN = false
	}
	if !initASN {
		if !mmdb.Verify(C.Path.ASN()) {
			log.Warnf("ASN invalid, remove and download")
			if err := os.Remove(C.Path.ASN()); err != nil {
				return fmt.Errorf("can't remove invalid ASN: %s", err.Error())
			}
			if err := mmdb.DownloadASN(C.Path.ASN()); err != nil {
				return fmt.Errorf("can't download ASN: %s", err.Error())
			}
		}
		initASN = true
	}
	return nil
}
