.SILENT: run

CYAN=\033[0;36m
RESET=\033[0m

pprint = echo -e "${CYAN}::>${RESET} ${1}"
completed = $(call pprint,Completed!)

run:
	$(call pprint, Run app...)
	go run ./cmd/main.go
	$(call completed)