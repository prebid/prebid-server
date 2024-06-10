# prebid-server-platform

...

## IaC Infrastructure
The `terraform.sh` script can be invoked to initialize, plan and apply terraforms. It is a
simple script designed to use a temporary directory to isolate terraform cli runs. If this is not sufficient,
the next step is to use a terraform container and mount specific project files and build artifacts. This
will guarantee a reproducable build on any machine.

### Script Options
`terraform --help` prints out usage information

#### AWS specific configuration
Environement is set using `-e`. You can set this to any name you desire, however, environment names for cmi-* accounts should match. e.g.
`-e development` in cmi-development1
`-e stage` in cmi-stage
`-e production` in cmi-production

@todo: automatically set these in stage and production accounts.

#### Variables
`-v`
In order to pass in project specific variables, analagous to `terraform -var='variable-name=variable-value', use `-v`.
As shown in example below, `-v variable-name=variable-value`

#### Testing/Planning
Simply add a `-t` flag to the run script to run `terraform init` && `terraform plan` only
```
./terraform.sh \
  -t \
  -w workspace-name \
  -e development \
  -p iac-profile-name \
  -v region=us-east-1
```

##### Deployment
To deploy, remove the `-t` flag will also invoke `terraform apply`
```
./terraform.sh \
  -w workspace-name \
  -e development \
  -p iac-profile-name \
  -v region=us-east-1
```
