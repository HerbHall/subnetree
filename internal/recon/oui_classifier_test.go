package recon

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestClassifyByManufacturer(t *testing.T) {
	tests := []struct {
		name         string
		manufacturer string
		want         models.DeviceType
	}{
		// Networking infrastructure.
		{"cisco router", "Cisco Systems, Inc.", models.DeviceTypeRouter},
		{"meraki router", "Meraki LLC", models.DeviceTypeRouter},
		{"mikrotik router", "MikroTik", models.DeviceTypeRouter},
		{"netgear router", "NETGEAR", models.DeviceTypeRouter},
		{"tp-link router", "TP-LINK TECHNOLOGIES CO.,LTD.", models.DeviceTypeRouter},
		{"d-link router", "D-Link Corporation", models.DeviceTypeRouter},
		{"linksys router", "Linksys", models.DeviceTypeRouter},
		{"asus router", "ASUSTek COMPUTER INC.", models.DeviceTypeRouter},

		// Access points.
		{"ubiquiti ap", "Ubiquiti Inc.", models.DeviceTypeAccessPoint},
		{"eero ap", "eero inc.", models.DeviceTypeAccessPoint},

		// Switches.
		{"aruba switch", "Aruba Networks", models.DeviceTypeSwitch},
		{"ruckus switch", "Ruckus Wireless", models.DeviceTypeSwitch},
		{"juniper switch", "Juniper Networks", models.DeviceTypeSwitch},
		{"hp networking switch", "HP Networking", models.DeviceTypeSwitch},
		{"hpe switch", "Hewlett Packard Enterprise", models.DeviceTypeSwitch},

		// Cameras.
		{"ring camera", "Ring LLC", models.DeviceTypeCamera},
		{"hikvision camera", "Hangzhou Hikvision Digital Technology", models.DeviceTypeCamera},
		{"dahua camera", "Zhejiang Dahua Technology", models.DeviceTypeCamera},
		{"reolink camera", "Reolink Innovation", models.DeviceTypeCamera},
		{"amcrest camera", "Amcrest Technologies", models.DeviceTypeCamera},
		{"wyze camera", "Wyze Labs", models.DeviceTypeCamera},

		// Printers.
		{"brother printer", "Brother Industries, Ltd.", models.DeviceTypePrinter},
		{"canon printer", "Canon Inc.", models.DeviceTypePrinter},
		{"epson printer", "Seiko Epson Corporation", models.DeviceTypePrinter},
		{"lexmark printer", "Lexmark International", models.DeviceTypePrinter},
		{"xerox printer", "Xerox Corporation", models.DeviceTypePrinter},
		{"ricoh printer", "Ricoh Company, Ltd.", models.DeviceTypePrinter},

		// NAS.
		{"synology nas", "Synology Incorporated", models.DeviceTypeNAS},
		{"qnap nas", "QNAP Systems, Inc.", models.DeviceTypeNAS},
		{"western digital nas", "Western Digital", models.DeviceTypeNAS},
		{"readynas nas", "ReadyNAS", models.DeviceTypeNAS},

		// Mobile devices.
		{"samsung mobile", "Samsung Electronics Co.,Ltd", models.DeviceTypeMobile},
		{"oneplus mobile", "OnePlus Technology", models.DeviceTypeMobile},
		{"xiaomi mobile", "Xiaomi Communications", models.DeviceTypeMobile},
		{"huawei mobile", "HUAWEI TECHNOLOGIES CO.,LTD", models.DeviceTypeMobile},
		{"oppo mobile", "OPPO Electronics Corp.", models.DeviceTypeMobile},
		{"vivo mobile", "vivo Mobile Communication", models.DeviceTypeMobile},
		{"motorola mobile", "Motorola Mobility LLC", models.DeviceTypeMobile},
		{"lg mobile", "LG Electronics", models.DeviceTypeMobile},

		// IoT.
		{"sonos iot", "Sonos, Inc.", models.DeviceTypeIoT},
		{"roku iot", "Roku, Inc.", models.DeviceTypeIoT},
		{"amazon iot", "Amazon Technologies Inc.", models.DeviceTypeIoT},
		{"raspberry pi iot", "Raspberry Pi Foundation", models.DeviceTypeIoT},
		{"espressif iot", "Espressif Inc.", models.DeviceTypeIoT},
		{"philips iot", "Philips Lighting BV", models.DeviceTypeIoT},
		{"ikea iot", "IKEA of Sweden", models.DeviceTypeIoT},
		{"shelly iot", "Shelly Europe Ltd.", models.DeviceTypeIoT},

		// Desktops.
		{"apple desktop", "Apple, Inc.", models.DeviceTypeDesktop},
		{"dell desktop", "Dell Inc.", models.DeviceTypeDesktop},
		{"lenovo desktop", "Lenovo", models.DeviceTypeDesktop},
		{"hp inc desktop", "HP Inc.", models.DeviceTypeDesktop},
		{"hewlett packard desktop", "Hewlett Packard", models.DeviceTypeDesktop},
		{"microsoft desktop", "Microsoft Corporation", models.DeviceTypeDesktop},

		// Ambiguous vendor: HP defaults to desktop, not switch.
		{"hp ambiguous defaults to desktop", "HP", models.DeviceTypeUnknown},

		// Unknown / empty.
		{"empty manufacturer", "", models.DeviceTypeUnknown},
		{"unknown manufacturer", "Acme Corp", models.DeviceTypeUnknown},
		{"whitespace only", "   ", models.DeviceTypeUnknown},

		// Case insensitivity.
		{"case insensitive cisco", "CISCO SYSTEMS", models.DeviceTypeRouter},
		{"case insensitive ubiquiti", "ubiquiti networks", models.DeviceTypeAccessPoint},
		{"case insensitive synology", "SYNOLOGY INCORPORATED", models.DeviceTypeNAS},
		{"mixed case samsung", "sAmSuNg ElEcTrOnIcS", models.DeviceTypeMobile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyByManufacturer(tt.manufacturer)
			if got != tt.want {
				t.Errorf("ClassifyByManufacturer(%q) = %q, want %q", tt.manufacturer, got, tt.want)
			}
		})
	}
}
