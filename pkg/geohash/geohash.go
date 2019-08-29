package geohash

import (
	"github.com/pkg/errors"

	"github.com/mmcloughlin/geohash"
)

const (
	// Unfortunately not exposed by the geohash library, but this is the max it
	// supports
	maxCharsPrecision = 12
)

// ForProviderAndRegionWithPrecision is the same as ForProviderAndRegion but
// allows a precision to be specified.
func ForProviderAndRegionWithPrecision(provider string, region string, precision uint) (string, error) {
	regions, ok := coordinatesLookup[provider]
	if !ok {
		return "", errors.Errorf("could not find provider %s in lookup table", provider)
	}

	coords, ok := regions[region]
	if !ok {
		return "", errors.Errorf("could not find region %s in lookup table", region)
	}

	return geohash.EncodeWithPrecision(coords.lat, coords.lon, precision), nil
}

// ForProviderAndRegion returns the geographic location of the given region for
// the given provider in the geohash format with the default precision (the max
// of 12), or an error if the provider or region can't be found in our lookup
// table.
func ForProviderAndRegion(provider string, region string) (string, error) {
	return ForProviderAndRegionWithPrecision(provider, region, maxCharsPrecision)
}
