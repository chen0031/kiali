package kubernetes

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kiali/kiali/config"
)

// GetIstioDetails returns Istio details for a given namespace,
// on this version it collects the VirtualServices and DestinationRules defined for a namespace.
// If serviceName param is provided, it filters all the Istio objects pointing to a particular service.
// It returns an error on any problem.
func (in *IstioClient) GetIstioDetails(namespace string, serviceName string) (*IstioDetails, error) {
	var virtualServices, destinationRules []IstioObject
	var virtualServicesErr, destinationRulesErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		virtualServices, virtualServicesErr = in.GetVirtualServices(namespace, serviceName)
	}()

	go func() {
		defer wg.Done()
		destinationRules, destinationRulesErr = in.GetDestinationRules(namespace, serviceName)
	}()

	wg.Wait()

	var istioDetails = IstioDetails{}

	istioDetails.VirtualServices = virtualServices
	if virtualServicesErr != nil {
		return nil, virtualServicesErr
	}

	istioDetails.DestinationRules = destinationRules
	if destinationRulesErr != nil {
		return nil, destinationRulesErr
	}

	return &istioDetails, nil
}

// GetVirtualServices return all VirtualServices for a given namespace.
// If serviceName param is provided it will filter all VirtualServices having a host defined on a particular service.
// It returns an error on any problem.
func (in *IstioClient) GetVirtualServices(namespace string, serviceName string) ([]IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(virtualServices).Do().Get()
	if err != nil {
		return nil, err
	}
	virtualServiceList, ok := result.(*VirtualServiceList)
	if !ok {
		return nil, fmt.Errorf("%s/%s doesn't return a VirtualService list", namespace, serviceName)
	}

	virtualServices := make([]IstioObject, 0)
	for _, virtualService := range virtualServiceList.GetItems() {
		appendVirtualService := serviceName == ""
		routeProtocols := []string{"http", "tcp"}
		if !appendVirtualService && FilterByRoute(virtualService.GetSpec(), routeProtocols, serviceName, namespace, nil) {
			appendVirtualService = true
		}
		if appendVirtualService {
			virtualServices = append(virtualServices, virtualService.DeepCopyIstioObject())
		}
	}
	return virtualServices, nil
}

func (in *IstioClient) GetVirtualService(namespace string, virtualservice string) (IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(virtualServices).SubResource(virtualservice).Do().Get()
	if err != nil {
		return nil, err
	}

	virtualService, ok := result.(*VirtualService)
	if !ok {
		return nil, fmt.Errorf("%s/%s doesn't return a VirtualService object", namespace, virtualservice)
	}
	return virtualService.DeepCopyIstioObject(), nil
}

// GetGateways return all Gateways for a given namespace.
// It returns an error on any problem.
func (in *IstioClient) GetGateways(namespace string) ([]IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(gateways).Do().Get()
	if err != nil {
		return nil, err
	}
	gatewayList, ok := result.(*GatewayList)
	if !ok {
		return nil, fmt.Errorf("%s doesn't return a Gateway list", namespace)
	}

	gateways := make([]IstioObject, 0)
	for _, gateway := range gatewayList.GetItems() {
		gateways = append(gateways, gateway.DeepCopyIstioObject())
	}
	return gateways, nil
}

func (in *IstioClient) GetGateway(namespace string, gateway string) (IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(gateways).SubResource(gateway).Do().Get()
	if err != nil {
		return nil, err
	}

	gatewayObject, ok := result.(*Gateway)
	if !ok {
		return nil, fmt.Errorf("%s/%s doesn't return a Gateway object", namespace, gateway)
	}
	return gatewayObject.DeepCopyIstioObject(), nil
}

// GetServiceEntries return all ServiceEntry objects for a given namespace.
// It returns an error on any problem.
func (in *IstioClient) GetServiceEntries(namespace string) ([]IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(serviceentries).Do().Get()
	if err != nil {
		return nil, err
	}
	serviceEntriesList, ok := result.(*ServiceEntryList)
	if !ok {
		return nil, fmt.Errorf("%s doesn't return a ServiceEntry list", namespace)
	}

	serviceEntries := make([]IstioObject, 0)
	for _, serviceEntry := range serviceEntriesList.GetItems() {
		serviceEntries = append(serviceEntries, serviceEntry.DeepCopyIstioObject())
	}
	return serviceEntries, nil
}

func (in *IstioClient) GetServiceEntry(namespace string, serviceEntryName string) (IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(serviceentries).SubResource(serviceEntryName).Do().Get()
	if err != nil {
		return nil, err
	}

	serviceEntry, ok := result.(*ServiceEntry)
	if !ok {
		return nil, fmt.Errorf("%s/%v doesn't return a ServiceEntry object", namespace, serviceEntry)
	}
	return serviceEntry.DeepCopyIstioObject(), nil
}

// GetDestinationRules returns all DestinationRules for a given namespace.
// If serviceName param is provided it will filter all DestinationRules having a host defined on a particular service.
// It returns an error on any problem.
func (in *IstioClient) GetDestinationRules(namespace string, serviceName string) ([]IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(destinationRules).Do().Get()
	if err != nil {
		return nil, err
	}
	destinationRuleList, ok := result.(*DestinationRuleList)
	if !ok {
		return nil, fmt.Errorf("%s/%s doesn't return a DestinationRule list", namespace, serviceName)
	}

	destinationRules := make([]IstioObject, 0)
	for _, destinationRule := range destinationRuleList.Items {
		appendDestinationRule := serviceName == ""
		if host, ok := destinationRule.Spec["host"]; ok {
			if dHost, ok := host.(string); ok && FilterByHost(dHost, serviceName, namespace) {
				appendDestinationRule = true
			}
		}
		if appendDestinationRule {
			destinationRules = append(destinationRules, destinationRule.DeepCopyIstioObject())
		}
	}
	return destinationRules, nil
}

func (in *IstioClient) GetDestinationRule(namespace string, destinationrule string) (IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(destinationRules).SubResource(destinationrule).Do().Get()
	if err != nil {
		return nil, err
	}
	destinationRule, ok := result.(*DestinationRule)
	if !ok {
		return nil, fmt.Errorf("%s/%s doesn't return a DestinationRule object", namespace, destinationrule)
	}
	return destinationRule.DeepCopyIstioObject(), nil
}

// GetQuotaSpecs returns all QuotaSpecs objects for a given namespace.
// It returns an error on any problem.
func (in *IstioClient) GetQuotaSpecs(namespace string) ([]IstioObject, error) {
	result, err := in.istioConfigApi.Get().Namespace(namespace).Resource(quotaspecs).Do().Get()
	if err != nil {
		return nil, err
	}
	quotaSpecList, ok := result.(*QuotaSpecList)
	if !ok {
		return nil, fmt.Errorf("%s doesn't return a QuotaSpecList list", namespace)
	}

	quotaSpecs := make([]IstioObject, 0)
	for _, qs := range quotaSpecList.GetItems() {
		quotaSpecs = append(quotaSpecs, qs.DeepCopyIstioObject())
	}
	return quotaSpecs, nil
}

func (in *IstioClient) GetQuotaSpec(namespace string, quotaSpecName string) (IstioObject, error) {
	result, err := in.istioConfigApi.Get().Namespace(namespace).Resource(quotaspecs).SubResource(quotaSpecName).Do().Get()
	if err != nil {
		return nil, err
	}

	quotaSpec, ok := result.(*QuotaSpec)
	if !ok {
		return nil, fmt.Errorf("%s/%s doesn't return a QuotaSpec object", namespace, quotaSpecName)
	}
	return quotaSpec.DeepCopyIstioObject(), nil
}

// GetQuotaSpecBindings returns all QuotaSpecBindings objects for a given namespace.
// It returns an error on any problem.
func (in *IstioClient) GetQuotaSpecBindings(namespace string) ([]IstioObject, error) {
	result, err := in.istioConfigApi.Get().Namespace(namespace).Resource(quotaspecbindings).Do().Get()
	if err != nil {
		return nil, err
	}
	quotaSpecBindingList, ok := result.(*QuotaSpecBindingList)
	if !ok {
		return nil, fmt.Errorf("%s doesn't return a QuotaSpecBindingList list", namespace)
	}

	quotaSpecBindings := make([]IstioObject, 0)
	for _, qs := range quotaSpecBindingList.GetItems() {
		quotaSpecBindings = append(quotaSpecBindings, qs.DeepCopyIstioObject())
	}
	return quotaSpecBindings, nil
}

func (in *IstioClient) GetQuotaSpecBinding(namespace string, quotaSpecBindingName string) (IstioObject, error) {
	result, err := in.istioConfigApi.Get().Namespace(namespace).Resource(quotaspecbindings).SubResource(quotaSpecBindingName).Do().Get()
	if err != nil {
		return nil, err
	}

	quotaSpecBinding, ok := result.(*QuotaSpecBinding)
	if !ok {
		return nil, fmt.Errorf("%s/%s doesn't return a QuotaSpecBinding object", namespace, quotaSpecBindingName)
	}
	return quotaSpecBinding.DeepCopyIstioObject(), nil
}

// CheckVirtualService returns true if virtualService object is defined for the service.
// It returns false otherwise.
func CheckVirtualService(virtualService IstioObject, namespace string, serviceName string) bool {
	if virtualService == nil || virtualService.GetSpec() == nil || serviceName == "" {
		return false
	}
	if hosts, ok := virtualService.GetSpec()["hosts"]; ok {
		for _, host := range hosts.([]interface{}) {
			if FilterByHost(host.(string), serviceName, namespace) {
				return true
			}
		}
	}
	return false
}

// CheckVirtualServiceSubset returns true if virtualService object has defined a route on a service for any subset passed as parameter.
// It returns false otherwise.
// TODO: Is this used?
func CheckVirtualServiceSubset(virtualService IstioObject, namespace string, serviceName string, subsets []string) bool {
	if virtualService == nil || virtualService.GetSpec() == nil || subsets == nil {
		return false
	}
	routeProtocols := []string{"http", "tcp"}
	if len(subsets) > 0 && FilterByRouteAndSubset(virtualService.GetSpec(), routeProtocols, serviceName, namespace, subsets) {
		return true
	}
	return false
}

// GetDestinationRulesSubsets returns an array of subset names where a specific version is defined for a given service
func GetDestinationRulesSubsets(destinationRules []IstioObject, serviceName, version string) []string {
	cfg := config.Get()
	foundSubsets := make([]string, 0)
	for _, destinationRule := range destinationRules {
		if dHost, ok := destinationRule.GetSpec()["host"]; ok {
			if host, ok := dHost.(string); ok && FilterByHost(host, serviceName, destinationRule.GetObjectMeta().Namespace) {
				if subsets, ok := destinationRule.GetSpec()["subsets"]; ok {
					if dSubsets, ok := subsets.([]interface{}); ok {
						for _, subset := range dSubsets {
							if innerSubset, ok := subset.(map[string]interface{}); ok {
								subsetName := innerSubset["name"]
								if labels, ok := innerSubset["labels"]; ok {
									if dLabels, ok := labels.(map[string]interface{}); ok {
										if versionValue, ok := dLabels[cfg.IstioLabels.VersionLabelName]; ok && versionValue == version {
											foundSubsets = append(foundSubsets, subsetName.(string))
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return foundSubsets
}

// CheckDestinationRuleCircuitBreaker returns true if the destinationRule object includes a trafficPolicy configuration
// on connectionPool or outlierDetection.
// TrafficPolicy configuration can be defined at service level or per subset defined by a version.
// It returns false otherwise.
func CheckDestinationRuleCircuitBreaker(destinationRule IstioObject, namespace string, serviceName string, version string) bool {
	if destinationRule == nil || destinationRule.GetSpec() == nil {
		return false
	}
	cfg := config.Get()
	if dHost, ok := destinationRule.GetSpec()["host"]; ok {
		if host, ok := dHost.(string); ok && FilterByHost(host, serviceName, namespace) {
			if trafficPolicy, ok := destinationRule.GetSpec()["trafficPolicy"]; ok && checkTrafficPolicy(trafficPolicy) {
				return true
			}
			if version == "" {
				return false
			}
			if subsets, ok := destinationRule.GetSpec()["subsets"]; ok {
				if dSubsets, ok := subsets.([]interface{}); ok {
					for _, subset := range dSubsets {
						if innerSubset, ok := subset.(map[string]interface{}); ok {
							if trafficPolicy, ok := innerSubset["trafficPolicy"]; ok && checkTrafficPolicy(trafficPolicy) {
								if labels, ok := innerSubset["labels"]; ok {
									if dLabels, ok := labels.(map[string]interface{}); ok {
										if versionValue, ok := dLabels[cfg.IstioLabels.VersionLabelName]; ok && versionValue == version {
											return true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

// Helper method to check if a trafficPolicy has defined a connetionPool or outlierDetection element
func checkTrafficPolicy(trafficPolicy interface{}) bool {
	if trafficPolicy == nil {
		return false
	}
	if dTrafficPolicy, ok := trafficPolicy.(map[string]interface{}); ok {
		if _, ok := dTrafficPolicy["connectionPool"]; ok {
			return true
		}
		if _, ok := dTrafficPolicy["outlierDetection"]; ok {
			return true
		}
	}
	return false
}

func FilterByDestination(spec map[string]interface{}, namespace string, serviceName string, version string) bool {
	if spec == nil {
		return false
	}
	cfg := config.Get()
	if destination, ok := spec["destination"]; ok {
		dest, ok := destination.(map[string]interface{})
		if !ok {
			return false
		}
		if dNamespace, ok := dest["namespace"]; ok && dNamespace != namespace {
			return false
		}
		if dName, ok := dest["name"]; ok && dName != serviceName {
			return false
		}

		if dLabels, ok := dest["labels"]; ok && version != "" {
			if labels, ok := dLabels.(map[string]interface{}); ok {
				if versionValue, ok := labels[cfg.IstioLabels.VersionLabelName]; ok && versionValue == version {
					return true
				}
				return false
			}
		} else {
			// It has not labels defined, but destination is defined on whole service
			return true
		}
	}
	return false
}

func FilterByHost(host string, serviceName string, namespace string) bool {
	// Check single name
	if host == serviceName {
		return true
	}
	// Check service.namespace
	if host == fmt.Sprintf("%s.%s", serviceName, namespace) {
		return true
	}
	// Check the FQDN. <service>.<namespace>.svc
	if host == fmt.Sprintf("%s.%s.%s", serviceName, namespace, "svc") {
		return true
	}

	// Check the FQDN. <service>.<namespace>.svc.<zone>
	if host == fmt.Sprintf("%s.%s.%s", serviceName, namespace, config.Get().ExternalServices.Istio.IstioIdentityDomain) {
		return true
	}

	// Note, FQDN names are defined from Kubernetes registry specification [1]
	// [1] https://github.com/kubernetes/dns/blob/master/docs/specification.md

	return false
}

func FilterByRoute(spec map[string]interface{}, protocols []string, service string, namespace string, serviceEntries map[string]struct{}) bool {
	if len(protocols) == 0 {
		return false
	}
	for _, protocol := range protocols {
		if prot, ok := spec[protocol]; ok {
			if aHttp, ok := prot.([]interface{}); ok {
				for _, httpRoute := range aHttp {
					if mHttpRoute, ok := httpRoute.(map[string]interface{}); ok {
						if route, ok := mHttpRoute["route"]; ok {
							if aDestinationWeight, ok := route.([]interface{}); ok {
								for _, destination := range aDestinationWeight {
									if mDestination, ok := destination.(map[string]interface{}); ok {
										if destinationW, ok := mDestination["destination"]; ok {
											if mDestinationW, ok := destinationW.(map[string]interface{}); ok {
												if host, ok := mDestinationW["host"]; ok {
													if sHost, ok := host.(string); ok {
														if FilterByHost(sHost, service, namespace) {
															return true
														}
														if serviceEntries != nil {
															// We have ServiceEntry to check
															if _, found := serviceEntries[strings.ToLower(protocol)+sHost]; found {
																return true
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

func FilterByRouteAndSubset(spec map[string]interface{}, protocols []string, service string, namespace string, subsets []string) bool {
	if len(protocols) == 0 {
		return false
	}
	for _, protocol := range protocols {
		if prot, ok := spec[protocol]; ok {
			if aHttp, ok := prot.([]interface{}); ok {
				for _, httpRoute := range aHttp {
					if mHttpRoute, ok := httpRoute.(map[string]interface{}); ok {
						if route, ok := mHttpRoute["route"]; ok {
							if aDestinationWeight, ok := route.([]interface{}); ok {
								for _, destination := range aDestinationWeight {
									if mDestination, ok := destination.(map[string]interface{}); ok {
										if destinationW, ok := mDestination["destination"]; ok {
											if mDestinationW, ok := destinationW.(map[string]interface{}); ok {
												if host, ok := mDestinationW["host"]; ok {
													if sHost, ok := host.(string); ok {
														if FilterByHost(sHost, service, namespace) {
															// Check service + name is found on a route
															if subset, ok := mDestinationW["subset"]; ok {
																if sSubset, ok := subset.(string); ok {
																	for _, checkSubset := range subsets {
																		if sSubset == checkSubset {
																			return true
																		}
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

// ServiceEntryHostnames returns a list of hostnames defined in the ServiceEntries Specs. Key in the resulting map is the protocol (in lowercase) + hostname
// exported for test
func ServiceEntryHostnames(serviceEntries []IstioObject) map[string]struct{} {
	var empty struct{}
	hostnames := make(map[string]struct{})

	for _, v := range serviceEntries {
		if hostsSpec, found := v.GetSpec()["hosts"]; found {
			if hosts, ok := hostsSpec.([]interface{}); ok {
				// Seek the protocol
				if portsSpec, found := v.GetSpec()["ports"]; found {
					if ports, ok := portsSpec.(map[string]interface{}); ok {
						if proto, found := ports["protocol"]; found {
							if protocol, ok := proto.(string); ok {
								protocol = mapPortToVirtualServiceProtocol(protocol)
								for _, v := range hosts {
									hostnames[fmt.Sprintf("%v%v", protocol, v)] = empty
								}

							}
						}
					}
				}
			}
		}
	}
	return hostnames
}

// mapPortToVirtualServiceProtocol transforms Istio's Port-definitions' protocol names to VirtualService's protocol names
func mapPortToVirtualServiceProtocol(proto string) string {
	// http: HTTP/HTTP2/GRPC/ TLS-terminated-HTTPS and service entry ports using HTTP/HTTP2/GRPC protocol
	// tls: HTTPS/TLS protocols (i.e. with “passthrough” TLS mode) and service entry ports using HTTPS/TLS protocols.
	// tcp: everything else

	switch proto {
	case "HTTP":
		fallthrough
	case "HTTP2":
		fallthrough
	case "GRPC":
		return "http"
	case "HTTPS":
		fallthrough
	case "TLS":
		return "tls"
	default:
		return "tcp"
	}
}

// GatewayNames extracts the gateway names for easier matching
func GatewayNames(gateways []IstioObject) map[string]struct{} {
	var empty struct{}
	names := make(map[string]struct{})
	for _, v := range gateways {
		v := v
		names[v.GetObjectMeta().Name] = empty
	}
	return names
}

// ValidateVirtualServiceGateways checks all VirtualService gateways (except mesh, which is reserved word) and checks that they're found from the given list of gatewayNames
func ValidateVirtualServiceGateways(spec map[string]interface{}, gatewayNames map[string]struct{}) bool {
	if gatewaysSpec, found := spec["gateways"]; found {
		if gateways, ok := gatewaysSpec.([]interface{}); ok {
			for _, g := range gateways {
				if gate, ok := g.(string); ok {
					if _, found := gatewayNames[gate]; !found && gate != "mesh" {
						return false
					}
				}
			}
		}
	}
	// No gateways defined or all found
	return true
}
