package resources

//
// import (
// 	"encoding/json"
//  "github.com/containership/cloud-agent/internal/log"
//
// 	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
// )
//
// // CsRegistries defines the Containership Cloud CsRegistries resource
// type CsRegistries struct {
// 	cloudResource
// 	cache []containershipv3.RegistrySpec
// }
//
// // NewCsRegistries constructs a new CsRegistries
// func NewCsRegistries() *CsRegistries {
// 	return &CsRegistries{
// 		cloudResource: cloudResource{
// 			endpoint: "/organizations/{{.OrganizationID}}/registries",
// 		},
// 		cache: make([]containershipv3.RegistrySpec, 0),
// 	}
// }
//
// // Endpoint returns the Endpoint
// func (rs *CsRegistries) Endpoint() string {
// 	return rs.endpoint
// }
//
// // UnmarshalToCache take the json returned from containership api
// // and writes it to CsRegistries cache
// func (rs *CsRegistries) UnmarshalToCache(bytes []byte) error {
// 	// clear cache before updating it
// 	rs.cache = nil
// 	log.Println("CsRegistries UnmarshallToCache...")
// 	err := json.Unmarshal(bytes, &rs.cache)
// 	log.Printf("CsRegistries cache updated: %+v", rs.cache)
// 	return err
// }
