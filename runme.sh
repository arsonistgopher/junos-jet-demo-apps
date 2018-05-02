mkdir -p proto/{jnx_addr,prpd_common,bgp_route,jnx_base,auth,management}
protoc -I ./proto --go_out=plugins=grpc,Mmanagement_service.proto=../management,Mjnx_base_types.proto=../jnx_base,Mjnx_addr.proto=../jnx_addr,Mprpd_common.proto=../prpd_common:./proto/bgp_route ./proto/bgp_route_service.proto
protoc -I ./proto --go_out=plugins=grpc,Mmanagement_service.proto=../management,Mjnx_base_types.proto=../jnx_base,Mjnx_addr.proto=../jnx_addr,Mprpd_common.proto=../prpd_common:./proto/jnx_addr ./proto/jnx_addr.proto
protoc -I ./proto --go_out=plugins=grpc,Mmanagement_service.proto=../management,Mjnx_base_types.proto=../jnx_base,Mjnx_addr.proto=../jnx_addr,Mprpd_common.proto=../prpd_common:./proto/prpd_common ./proto/prpd_common.proto
protoc -I ./proto --go_out=plugins=grpc,Mmanagement_service.proto=../management,Mjnx_base_types.proto=../jnx_base,Mjnx_addr.proto=../jnx_addr,Mprpd_common.proto=../prpd_common:./proto/jnx_base ./proto/jnx_base_types.proto
protoc -I ./proto --go_out=plugins=grpc,Mmanagement_service.proto=../management,Mjnx_base_types.proto=../jnx_base,Mjnx_addr.proto=../jnx_addr,Mprpd_common.proto=../prpd_common:./proto/management ./proto/management_service.proto
