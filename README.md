**In development**

# terrafix

A tool fixes user's terraform configurations to match the targeting provider's schema.

## Introduction

This tool tries to make introducing of breaking changes into a terraform provider fearlessly, for both the provider developers and the users:

1. For provider developers: providing a way to define configuration migration logics for each resource, similar as how state migration did
2. For users: providing a CLI tool to automatically fix the user’s module(s), to match the schemas defined in the new version of the provider

## How

The full solution is composed of two parts:

1. `terrafix`: The CLI tool that is responsible to *understand* the user’s module(s). Interact with the provider (via provider functions for now), passing the current content of the HCL code (either the reference origins, or the resource definitions), together with the version of the belonging resource's schema. In case there is a terraform state available, it will also be passed to the provider. The terraform provider then updates the HCL content and return it back. The tool will then update the contents of the module(s) accordingly.
2. The terraform provider: The provider is supposed to implement the predefined provider functions, which are similar to the state migration functions, but for configurations. Helper functions are provided in the [terrafix-sdk](https://github.com/magodo/terrafix-sdk), to make it quite easy to opt in, for any [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework) based provider.
    
    The configuration update logic is in the provider for the same reason as the state migration, as the provider developer has the full knowledge about how that breaking change will affect the configuration.

The tool is using the same underlying libraries as the [terraform-ls](https://github.com/hashicorp/terraform-ls), which makes it has a full grasp of the module(s), including all the references scattered around.

The complete process about how to use the tool is described below:

1. Download the new version of the target provider
2. Run `terrafix` and specify the root module path, together with the path to the provider and the provider’s FQN (this makes the tool to target on that provider only)
3. The tool will then:
    1. Parse the module(s), with the *old* provider’s schema
    2. Call the provider function against the new provider, to update reference origins that target to resources belonging to that provider
    3. Update the contents of the modules (in memory) with the updated references
    4. Parse the module(s) again, with the *old* provider’s schema (this still succeeds as the schema validation only applies to the *left hand side*)
    5. Call the provider function against the new provider, to update resource definitions that belonging to that provider
    6. Update the contents of the module(s) (in memory) with the updated definitions
    7. Write out the module(s) from memory to either the original location, or another location (for inspection)
4. To this point, the configurations conform to the new provider’s schema. While the state and the provider in use is still for the old version. The user is then supposed to upgrade the provider via `terraform init -upgrade`. Finally, run `terraform plan` to trigger the state migration, and verify both the state and configurations are in the good shape.

## Use Cases

### Provider Upgrade

When there is a breaking change made to the provider, the provider developer is supposed to also implement the state migration, together with the config migration, in the same PR.

The users can use this tool to fearlessly upgrade the provider, with the configuration and state migrated automatically.

### Ad-hoc Batch Modifications

We've provided a dummy provider codebase [terraform-provider-fix](https://github.com/magodo/terraform-provider-fix), that users can clone and implement arbitrary config fix logics as they wish and run the `terrafix` tool against this provider. This allows users to programmably refactor the resource definitions and their references consistently.

## Notes

- Currently, the configuration fix only scopes at a single resource. Breaking changes that split/merge resources are not supported. These requires an overall picture of the module(s), that isn’t a good fit as the current design of the configuration/state migration residing at the provider side.
- Some breaking changes maps the value of an attribute to a different value sets. If the original value is not a literal value, the mapped new value is only known at run-time. Fortunately, since we will also provide the state of the resource to the provider for the config upgrade, which can be used to map to the new value. Alternatively, the provider can define this transformation in a new provider function, and take the call to this function as the new value.
- Reference that contain index (due to the use of `for_each` or `count`) won’t be recognized for now. This probably is a bug in the underlying [`github.com/hashicorp/hcl-lang`](http://github.com/hashicorp/hcl-lang) module.
- As is mentioned, the config definition request will send the terraform state of the resource to the provider, as long as there is NOT any index use along the address to this resource (due to the use of `for_each` or `count`, for either the resource or the module). The rationale behind this is that the tool aims to update the configuration, which is one piece of code, no matter it is a single instance, or a collection of instances. For the latter case, the state is meaningless to be consulted.

## Examples

### terraform-provider-fix based

Checkout the [README.md](https://github.com/magodo/terraform-provider-fix/) for a dummy example targeting the `terraform-provider-null` provider.

### terraform-provider-azurerm based

Checkout the [commit](https://github.com/magodo/terraform-provider-azurerm/tree/e580a64e42dd1e4a1c4fd1a7a11b4eed23d90730) for the `terraform-provider-azurerm`, based on its v4.2.0. This commit includes the following changes:

1. Introduces breaking changes to the `azurerm_virtual_network` resource:
    1. Rename the attribute `guid` to `uuid`
    2. Extends the attribute `location` (a string) to `locations` (a list of strings)
    3. Update the schema version from 0 to 1
2. Introduces one more breaking change to the `azurerm_virtual_network` resource:
    1. Changes the attribute `guid` from computed only to required
    2. Update the schema version from 1 to 2
3. Opt in `terrafix` functionality by implementing the provider functions based on `terrafix-sdk`

The goal is to have a smooth provider upgrade experience from v4.2.0, to this commit. Based on the module at https://github.com/magodo/terrafix/tree/b291406afdbb9002e63ed0b0bebb84a7a02d3d01/internal/ctrl/testdata/module (with the state).

You can try this example out via following the steps mentioned in the **How** section.
