.PHONY: serve watch-serve db-reset test docker-deps docker-build docker-run docker deploy-deps deploy check-to

serve:
	go install github.com/samertm/githubstreaks
	githubstreaks

watch-serve:
	$(shell while true; do $(MAKE) serve & PID=$$! ; echo $$PID ; inotifywait --exclude ".git" -r -e close_write . ; kill $$PID ; done)

db-reset:
	psql -h localhost -U ghs -c "drop schema public cascade"
	psql -h localhost -U ghs -c "create schema public"

test:
	go test $(ARGS) ./...

docker-deps:
	$(MAKE) -C postgres-docker docker-build
	$(MAKE) -C postgres-docker run-prod

docker-build:
	docker build -t ghs .

docker-run:
	docker start ghs-db # Did you run 'make docker-deps'?
	-docker top ghs-app && docker rm -f ghs-app
	docker run -d -p 8111:8000 --name ghs-app --link ghs-db:ghs-db ghs # Did you run 'make docker-build?'

docker: docker-build docker-run

# Must specify TO.
deploy-deps: check-to
	rsync -azP . $(TO):~/githubstreaks
	ssh $(TO) 'cd ~/githubstreaks && make docker-deps'

# Must specify TO.
deploy: check-to
	rsync -azP . $(TO):~/githubstreaks
	ssh $(TO) 'cd ~/githubstreaks && make docker'

check-to:
	ifndef TO
	    $(error TO is undefined)
	endif
