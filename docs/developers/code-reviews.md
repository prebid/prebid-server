# Code Reviews

## Standards
Anyone is free to review and comment on any [open pull requests](https://github.com/prebid/prebid-server/pulls).

All pull requests must be reviewed and approved by at least one [core member](https://github.com/orgs/prebid/teams/core/members) before merge.

Very small pull requests may be merged with just one review if they:

1. Do not change the public API.
2. Have low risk of bugs, in the opinion of the reviewer.
3. Introduce no new features, or impact the code architecture.

Larger pull requests must meet at least one of the following two additional requirements.

1. Have a second approval from a core member
2. Be open for 5 business days with no new changes requested.

## Process

New pull requests should be [assigned](https://help.github.com/articles/assigning-issues-and-pull-requests-to-other-github-users/) to a core member for review within 3 business days of being opened.
That person should either approve the changes or request changes within 4 business days of being assigned.
If they're too busy, they should assign it to someone else who can review it within that timeframe.

If the changes are small, that member can merge the PR once the changes are complete. Otherwise, they should
assign the pull request to another member for a second review.

The pull request can then be merged whenever the second reviewer approves, or if 5 business days pass with no farther
changes requested by anybody, whichever comes first.


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
