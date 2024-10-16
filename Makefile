init:
	terraform -chdir=internal/ctrl/testdata/module init
	terraform -chdir=internal/state/testdata/simple_module init
	terraform -chdir=internal/state/testdata/remote_module init
	terraform -chdir=internal/state/testdata/nested_modules init
