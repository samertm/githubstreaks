.PHONY: serve watch-serve db-reset psql remote-psql test docker-deps docker-build docker-run docker deploy-deps deploy

serve:
	go install github.com/samertm/githubstreaks
	githubstreaks

watch-serve:
	$(shell while true; do $(MAKE) serve & PID=$$! ; echo $$PID ; inotifywait --exclude ".git" -r -e close_write . ; kill $$PID ; done)

db-reset:
	psql -h localhost -U ghs -c "drop schema public cascade"
	psql -h localhost -U ghs -c "create schema public"

psql:
	psql -h localhost -U ghs

remote-psql:
	ssh -t $(TO) 'docker exec -it ghs-db bash -c "psql -U ghs"' # -t means ssh in tty mode.

test:
	go test -v $(ARGS) ./...

docker-deps:
	$(MAKE) -C postgres-docker docker-build
	$(MAKE) -C postgres-docker run-prod

docker-build:
	docker build -t ghs .

docker-run:
	docker start ghs-db # Did you run 'make docker-deps'?
	-docker top ghs-app && docker rm -f ghs-app
	docker run -d -p 8222:8000 --name ghs-app --link ghs-db:postgres ghs # Did you run 'make docker-build?'

docker: docker-build docker-run

# Must specify TO.
deploy-deps:
	rsync -azP . $(TO):~/githubstreaks
	ssh $(TO) 'cd ~/githubstreaks && make docker-deps'

# Must specify TO.
deploy:
	rsync -azP . $(TO):~/githubstreaks
	ssh $(TO) 'cd ~/githubstreaks && make docker'

