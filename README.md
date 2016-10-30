# GOC-Proxy

GOC-Proxy is a dynamic reverse proxy and load balancer that uses Consul to generate routes to your services. 
Each time the Consul catalog changes, GOC-Proxy updates it's service registry and reroutes the HTTP calls to the new service instances. 
GOC-Proxy is not designed to be an Internet facing reverse proxy, it's main purpose is to route traffic between microservices behind the firewall.
