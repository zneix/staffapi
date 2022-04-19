build:
	@cd cmd/staffapi && go build "${@:2}"

run: build
	@if test -f ./env.sh; then source ./env.sh; fi \
	&& cmd/staffapi/staffapi
