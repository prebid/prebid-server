# Code Reviews

## Priorities

Code reviews should focus on things which cannot be validated by machines.

Some examples include:

- Can we improve the user's experience in any way?
- Have the relevant [docs](..) been added or updated? If not, add the `needs docs` label.
- Do you believe that the code works by looking at the unit tests? If not, suggest more tests until you do!
- Is the motivation behind these changes clear? If not, there must be [an issue](https://github.com/prebid/prebid-server/issues) explaining it. Are there better ways to achieve those goals?
- Does the code use any global, mutable state? [Inject dependencies](https://en.wikipedia.org/wiki/Dependency_injection) instead!
- Can the code be organized into smaller, more modular pieces?
- Is there dead code which can be deleted? Or TODO comments which should be resolved?
- Look for code used by other adapters. Encourage adapter submitter to utilize common code. 
