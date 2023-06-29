# PR-comment-filter

Designed to be an entrypoint task for GitHub webhook triggers based on PR comments. `pr-comment-filter` will check the comment text for recognised triggers (e.g. `/run my-pipeline`) and generate individual `PipelineRun` resources for each.

The generated `PipelineRun` will have details about the PR passed through as params as well as any arguments added to the trigger comment.

## Trigger format

All triggers must follow the following format:

```
/run ${PIPELINE_NAME} [KEY=val ...]
```

Examples:

```
/run build-and-publish
/run test-cluster-create PRIVATE_NETWORK=true
/run test-cluster-create PREVIOUS_VERSION=1.2.6
/run test-cluster-upgrade PRIVATE_NETWORK=false PREVIOUS_VERSION=1.2.6
/run hold wait-for-tests
/run help NAMESPACE=foo-bar test-cluster-create
```

Some notes:

* Multiple triggers can be defined in a single comment but must each be on their own line
* Triggers must start the line with `/run ` and cannot be placed mid-sentance
* Arguments are optional and in the format of either:
  * `KEY=value` where the key must be all uppercase and is used as the key/val pair of environment variables
  * space seperated words - these are treated as "positional arguments" and added as the `POS_ARGS` env var with the values comma seperated
* Multiple arguments can be provided as long as they all appear on the same line as the trigger
* Argument values with spaces in them is not currently supported
* A Pipeline from a specific namespace can be run by specifying a `NAMESPACE=xxx` argument along with the `/run` trigger line
* If a user provided namespace isn't provided the pipeline will first be looked for in a namespace matching the repo name and if not found then default back to the `tekton-pipelines` namespace.
