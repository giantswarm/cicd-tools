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
```

Some notes:

* Multiple triggers can be defined in a single comment but must each be on their own line
* Triggers must start the line with `/run ` and cannot be placed mid-sentance
* Arguments are optional and in the format of `KEY=value` where the key must be all uppercase
* Multiple arguments can be provided as long as they all appear on the same line as the trigger
* Argument values with spaces in them is not currently supported
