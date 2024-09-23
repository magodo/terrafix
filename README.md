**In development**

# terrafix

A tool to fix user's terraform configurations to match the targeting provider's schema.

## TODO

- [ ] Correctly pass the schema version (currently it is always 0)
- [ ] Fixer: provider SDK for easier onboarding
- [ ] Fixer: non-provider implementation, TBD
- [ ] FixConfigDefinition request supports sending the state
- [ ] hcl-lang doesn't understand reference origins targeting to instances from `count`, `for_each`
- [ ] (LT) Provider block fix capability
