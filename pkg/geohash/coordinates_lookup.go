package geohash

// TODO the content of this file should be autogenerated or fetched and stored
// in memory on boot. We should consider storing geohashes (at max precision)
// directly instead of computing them on the fly.

type coordinates struct {
	lat float64
	lon float64
}

var coordinatesLookup = map[string]map[string]coordinates{
	"aws": {
		"ap-south-1": coordinates{
			lat: 19.076,
			lon: 72.8777,
		},
		"ap-northeast-1": coordinates{
			lat: 35.6762,
			lon: 139.6503,
		},
		"ap-northeast-2": coordinates{
			lat: 37.5665,
			lon: 126.978,
		},
		"ap-southeast-1": coordinates{
			lat: 1.3521,
			lon: 103.8198,
		},
		"ap-southeast-2": coordinates{
			lat: -33.8688,
			lon: 151.2093,
		},
		"ca-central-1": coordinates{
			lat: 45.5017,
			lon: -73.5673,
		},
		"eu-central-1": coordinates{
			lat: 50.1109,
			lon: 8.6821,
		},
		"eu-west-1": coordinates{
			lat: 53.3498,
			lon: -6.2603,
		},
		"eu-west-2": coordinates{
			lat: 51.5074,
			lon: -0.1278,
		},
		"eu-west-3": coordinates{
			lat: 48.8566,
			lon: 2.3522,
		},
		"sa-east-1": coordinates{
			lat: -23.5505,
			lon: -46.6333,
		},
		"us-east-1": coordinates{
			lat: 39.0438,
			lon: -77.4874,
		},
		"us-east-2": coordinates{
			lat: 39.9612,
			lon: -82.9988,
		},
		"us-west-1": coordinates{
			lat: 37.3541,
			lon: -121.9552,
		},
		"us-west-2": coordinates{
			lat: 45.8526,
			lon: -119.672,
		},
	},
	"azure": {
		"australiacentral": coordinates{
			lat: -35.2809,
			lon: 149.13,
		},
		"australiacentral2": coordinates{
			lat: -35.2809,
			lon: 149.13,
		},
		"australiaeast": coordinates{
			lat: -31.2532,
			lon: 146.9211,
		},
		"australiasoutheast": coordinates{
			lat: -37.4713,
			lon: 144.7852,
		},
		"brazilsouth": coordinates{
			lat: -23.5505,
			lon: -46.6333,
		},
		"canadacentral": coordinates{
			lat: 43.6532,
			lon: -79.3832,
		},
		"canadaeast": coordinates{
			lat: 46.8139,
			lon: -71.208,
		},
		"centralindia": coordinates{
			lat: 18.5204,
			lon: 73.8567,
		},
		"centralus": coordinates{
			lat: 41.5868,
			lon: -93.625,
		},
		"chinaeast": coordinates{
			lat: 31.2304,
			lon: 121.4737,
		},
		"chinaeast2": coordinates{
			lat: 31.2304,
			lon: 121.4737,
		},
		"chinanorth": coordinates{
			lat: 39.9042,
			lon: 116.4074,
		},
		"chinanorth2": coordinates{
			lat: 39.9042,
			lon: 116.4074,
		},
		"eastasia": coordinates{
			lat: 22.3193,
			lon: 114.1694,
		},
		"eastus": coordinates{
			lat: 38.586,
			lon: -79.395,
		},
		"eastus2": coordinates{
			lat: 38.586,
			lon: -79.395,
		},
		"francecentral": coordinates{
			lat: 48.8566,
			lon: 2.3522,
		},
		"francesouth": coordinates{
			lat: 43.2965,
			lon: 5.3698,
		},
		"germanycentral": coordinates{
			lat: 50.1109,
			lon: 8.6821,
		},
		"germanynortheast": coordinates{
			lat: 52.1205,
			lon: 11.6276,
		},
		"japaneast": coordinates{
			lat: 35.6762,
			lon: 139.6503,
		},
		"japanwest": coordinates{
			lat: 34.6937,
			lon: 135.5023,
		},
		"koreacentral": coordinates{
			lat: 37.5665,
			lon: 126.978,
		},
		"koreasouth": coordinates{
			lat: 35.1796,
			lon: 129.0756,
		},
		"northcentralus": coordinates{
			lat: 41.8781,
			lon: -87.6298,
		},
		"northeurope": coordinates{
			lat: 53.3498,
			lon: -6.2603,
		},
		"southeastasia": coordinates{
			lat: 1.3521,
			lon: 103.8198,
		},
		"southafricanorth": coordinates{
			lat: -26.2041,
			lon: 28.0473,
		},
		"southafricawest": coordinates{
			lat: -33.9249,
			lon: 18.4241,
		},
		"southcentralus": coordinates{
			lat: 29.4241,
			lon: -98.4936,
		},
		"southindia": coordinates{
			lat: 13.0827,
			lon: 80.2707,
		},
		"uaecentral": coordinates{
			lat: 24.4539,
			lon: 54.3773,
		},
		"uaenorth": coordinates{
			lat: 25.2048,
			lon: 55.2708,
		},
		"uksouth": coordinates{
			lat: 51.5074,
			lon: -0.1278,
		},
		"ukwest": coordinates{
			lat: 51.4816,
			lon: -3.1791,
		},
		"westcentralus": coordinates{
			lat: 41.14,
			lon: -104.8202,
		},
		"westeurope": coordinates{
			lat: 52.3667,
			lon: 4.8945,
		},
		"westindia": coordinates{
			lat: 19.076,
			lon: 72.8777,
		},
		"westus": coordinates{
			lat: 37.3688,
			lon: -122.0363,
		},
		"westus2": coordinates{
			lat: 47.2343,
			lon: -119.8526,
		},
	},
	"digitalocean": {
		"ams3": coordinates{
			lat: 52.3667,
			lon: 4.8945,
		},
		"blr1": coordinates{
			lat: 12.9716,
			lon: 77.5946,
		},
		"fra1": coordinates{
			lat: 50.1109,
			lon: 8.6821,
		},
		"lon1": coordinates{
			lat: 51.5074,
			lon: -0.1278,
		},
		"nyc1": coordinates{
			lat: 40.7128,
			lon: -74.006,
		},
		"nyc3": coordinates{
			lat: 40.7128,
			lon: -74.006,
		},
		"sfo2": coordinates{
			lat: 37.7749,
			lon: -122.4194,
		},
		"sgp1": coordinates{
			lat: 1.3521,
			lon: 103.8198,
		},
		"tor1": coordinates{
			lat: 43.6532,
			lon: -79.3832,
		},
	},
	"gce": {
		"asia-east1": coordinates{
			lat: 24.0518,
			lon: 120.5161,
		},
		"asia-northeast1": coordinates{
			lat: 35.6895,
			lon: 139.6917,
		},
		"asia-south1": coordinates{
			lat: 19.076,
			lon: 72.8777,
		},
		"asia-southeast1": coordinates{
			lat: 1.3404,
			lon: 103.709,
		},
		"australia-southeast1": coordinates{
			lat: 33.8688,
			lon: 151.2093,
		},
		"europe-north1": coordinates{
			lat: 60.5693,
			lon: 27.1878,
		},
		"europe-west1": coordinates{
			lat: 50.4491,
			lon: 3.8184,
		},
		"europe-west2": coordinates{
			lat: 51.5074,
			lon: -0.1278,
		},
		"europe-west3": coordinates{
			lat: 50.1109,
			lon: 8.6821,
		},
		"europe-west4": coordinates{
			lat: 53.4386,
			lon: 6.8355,
		},
		"northamerica-northeast1": coordinates{
			lat: 45.5017,
			lon: -73.5673,
		},
		"southamerica-east1": coordinates{
			lat: -23.5505,
			lon: -46.6333,
		},
		"us-central1": coordinates{
			lat: 41.2619,
			lon: -95.8608,
		},
		"us-east1": coordinates{
			lat: 33.196,
			lon: -80.0131,
		},
		"us-east4": coordinates{
			lat: 39.0438,
			lon: -77.4874,
		},
		"us-west1": coordinates{
			lat: 45.5946,
			lon: -121.1787,
		},
		"us-west2": coordinates{
			lat: 34.0522,
			lon: -118.2437,
		},
	},
	"packet": {
		"ams1": coordinates{
			lat: 52.3667,
			lon: 4.8945,
		},
		"dfw2": coordinates{
			lat: 32.7767,
			lon: -96.797,
		},
		"ewr1": coordinates{
			lat: 40.8653,
			lon: -74.4174,
		},
		"iad1": coordinates{
			lat: 39.0438,
			lon: -77.4874,
		},
		"nrt1": coordinates{
			lat: 35.6762,
			lon: 139.6503,
		},
		"sin1": coordinates{
			lat: 1.3521,
			lon: 103.8198,
		},
		"sjc1": coordinates{
			lat: 37.3688,
			lon: -122.0363,
		},
		"syd1": coordinates{
			lat: -33.8688,
			lon: 151.2093,
		},
	},
}
