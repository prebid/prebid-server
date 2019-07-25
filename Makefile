# Makefile
APP=prebid-server


PROJECT-TYPE=other
CONTAINER_PORTS=-p 5000:8000/tcp -p 18080:8080/tcp
BUILD_CREDENTIALS=true

# overrides
.ME-postup=off
.ME-test=off


# Include common makefile
-include microservices-ext/make/Makefile-common.mk

# Or get it, if it's not there
GITURL:=$(shell git remote -v | awk '{sub("8/.*", "8/", $$2); print $$2}' | head -1)
$(.ME-ext)microservices-ext:
	git clone -q https://github.com/spilgames/microservices-ext
	-@test "`grep microservices-ext .gitignore`" || echo "microservices-ext/" >> .gitignore
	@make $(MAKECMDGOALS)


postup::
	$(CONSUL) wait

test:
	@curl -i 'localhost:5000' | grep 200
