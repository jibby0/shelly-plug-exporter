package shellyplug

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/webdevops/shelly-plug-exporter/discovery"
	"github.com/webdevops/shelly-plug-exporter/shellyprober"
)

type (
	shellyGen2ConfigValue struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}
)

func (sp *ShellyPlug) collectFromTargetGen2(target discovery.DiscoveryTarget, logger *log.Entry, infoLabels, targetLabels prometheus.Labels) {
	sp.prometheus.info.With(infoLabels).Set(1)

	shellyProber := shellyprober.ShellyProberGen2{
		Target: target,
		Client: sp.client,
		Ctx:    sp.ctx,
		Cache:  globalCache,
	}

	if shellyConfig, err := shellyProber.GetShellyConfig(); err == nil {
		// systemStatus
		if result, err := shellyProber.GetSysStatus(); err == nil {
			sp.prometheus.sysUnixtime.With(targetLabels).Set(float64(result.Unixtime))
			sp.prometheus.sysUptime.With(targetLabels).Set(float64(result.Uptime))
			sp.prometheus.sysMemTotal.With(targetLabels).Set(float64(result.RAMSize))
			sp.prometheus.sysMemFree.With(targetLabels).Set(float64(result.RAMFree))
			sp.prometheus.sysFsSize.With(targetLabels).Set(float64(result.FsSize))
			sp.prometheus.sysFsFree.With(targetLabels).Set(float64(result.FsFree))
			sp.prometheus.restartRequired.With(targetLabels).Set(boolToFloat64(result.RestartRequired))
		} else {
			logger.Errorf(`failed to decode sysConfig: %v`, err)
		}

		// wifiStatus
		if result, err := shellyProber.GetWifiStatus(); err == nil {
			wifiLabels := copyLabelMap(targetLabels)
			wifiLabels["ssid"] = result.Ssid
			sp.prometheus.wifiRssi.With(wifiLabels).Set(float64(result.Rssi))
		} else {
			logger.Errorf(`failed to decode wifiStatus: %v`, err)
		}

		for configName, configValue := range shellyConfig {
			switch {
			// switch
			case strings.HasPrefix(configName, "switch:"):
				if configData, err := decodeShellyConfigValueToItem(configValue); err == nil {
					if result, err := shellyProber.GetSwitchStatus(configData.Id); err == nil {
						switchLabels := copyLabelMap(targetLabels)
						switchLabels["id"] = fmt.Sprintf("switch:%d", configData.Id)
						switchLabels["name"] = configData.Name

						switchOnLabels := copyLabelMap(switchLabels)
						switchOnLabels["source"] = result.Source

						sp.prometheus.switchOn.With(switchOnLabels).Set(boolToFloat64(result.Output))

						powerUsageLabels := copyLabelMap(targetLabels)
						powerUsageLabels["id"] = fmt.Sprintf("switch:%d", configData.Id)
						powerUsageLabels["name"] = configData.Name
						sp.prometheus.powerCurrent.With(powerUsageLabels).Set(result.Current)
						sp.prometheus.powerTotal.With(powerUsageLabels).Set(result.Apower)
					} else {
						logger.Errorf(`failed to decode switchStatus: %v`, err)
					}
				}
			// em
			case strings.HasPrefix(configName, "em:"):
				if configData, err := decodeShellyConfigValueToItem(configValue); err == nil {
					if result, err := shellyProber.GetEmStatus(configData.Id); err == nil {
						// phase A
						powerUsageLabels := copyLabelMap(targetLabels)
						powerUsageLabels["id"] = fmt.Sprintf("em:%d:A", configData.Id)
						powerUsageLabels["name"] = configData.Name
						sp.prometheus.powerCurrent.With(powerUsageLabels).Set(result.ACurrent)
						sp.prometheus.powerApparentCurrent.With(powerUsageLabels).Set(result.AAprtPower)
						sp.prometheus.powerTotal.With(powerUsageLabels).Set(result.AActPower)
						sp.prometheus.powerFactor.With(powerUsageLabels).Set(result.APf)
						sp.prometheus.powerFrequency.With(powerUsageLabels).Set(result.AFreq)
						sp.prometheus.powerVoltage.With(powerUsageLabels).Set(result.AVoltage)

						// phase B
						powerUsageLabels = copyLabelMap(targetLabels)
						powerUsageLabels["id"] = fmt.Sprintf("em:%d:B", configData.Id)
						powerUsageLabels["name"] = configData.Name
						sp.prometheus.powerCurrent.With(powerUsageLabels).Set(result.BCurrent)
						sp.prometheus.powerApparentCurrent.With(powerUsageLabels).Set(result.BAprtPower)
						sp.prometheus.powerTotal.With(powerUsageLabels).Set(result.BActPower)
						sp.prometheus.powerFactor.With(powerUsageLabels).Set(result.BPf)
						sp.prometheus.powerFrequency.With(powerUsageLabels).Set(result.BFreq)
						sp.prometheus.powerVoltage.With(powerUsageLabels).Set(result.BVoltage)

						// phase C
						powerUsageLabels = copyLabelMap(targetLabels)
						powerUsageLabels["id"] = fmt.Sprintf("em:%d:C", configData.Id)
						powerUsageLabels["name"] = configData.Name
						sp.prometheus.powerCurrent.With(powerUsageLabels).Set(result.CCurrent)
						sp.prometheus.powerApparentCurrent.With(powerUsageLabels).Set(result.CAprtPower)
						sp.prometheus.powerTotal.With(powerUsageLabels).Set(result.CActPower)
						sp.prometheus.powerFactor.With(powerUsageLabels).Set(result.CPf)
						sp.prometheus.powerFrequency.With(powerUsageLabels).Set(result.CFreq)
						sp.prometheus.powerVoltage.With(powerUsageLabels).Set(result.CVoltage)

						// phase C
						powerUsageLabels = copyLabelMap(targetLabels)
						powerUsageLabels["id"] = "em:total"
						powerUsageLabels["name"] = configData.Name
						sp.prometheus.powerCurrent.With(powerUsageLabels).Set(result.TotalCurrent)
						sp.prometheus.powerTotal.With(powerUsageLabels).Set(result.TotalActPower)
					} else {
						logger.Errorf(`failed to decode switchStatus: %v`, err)
					}
				}
			// temperatureSensor
			case strings.HasPrefix(configName, "temperature:"):
				if configData, err := decodeShellyConfigValueToItem(configValue); err == nil {
					if result, err := shellyProber.GetTemperatureStatus(configData.Id); err == nil {
						tempLabels := copyLabelMap(targetLabels)
						tempLabels["id"] = fmt.Sprintf("sensor:%d", configData.Id)
						tempLabels["name"] = configData.Name

						sp.prometheus.temp.With(tempLabels).Set(result.TC)
					} else {
						logger.Errorf(`failed to decode temperatureStatus: %v`, err)
					}
				}
			}
		}
	} else {
		logger.Errorf(`failed to fetch status: %v`, err)
		if discovery.ServiceDiscovery != nil {
			discovery.ServiceDiscovery.MarkTarget(target.Address, discovery.TargetUnhealthy)
		}
	}
}

func decodeShellyConfigValueToItem(val interface{}) (shellyGen2ConfigValue, error) {
	ret := shellyGen2ConfigValue{}

	data, err := json.Marshal(val)
	if err != nil {
		return ret, err
	}

	err = json.Unmarshal(data, &ret)
	return ret, err
}
