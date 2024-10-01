# What's New

## Version 2.0.0 (01. October 2024) - Laravel Support

With this release, we support **Laravel** framework as first-class framework:

- Download database dump from Laravel applications (encrypted)
- Download public resources from Laravel applications
- Download private resources from Laravel applications (encrypted)

## Version 1.3.0 (24. July 2023) - work with different PHP versions

With this release, we auto-detect the correct PHP version to use during `synco serve`.

If needed, you can override the PHP version by setting the `PHP` environment variable to the PHP interpreter,
e.g.:

```bash
export PHP=/my/special/path/to/php
# synco serve command as usual
```

## Version 1.2.0 (01. April 2023) - Big improvements for large instances of Neos/Flow

This release features numerous important quality-of-life improvements for large instances of Neos
and Flow with big databases and/or many persistent resources (=files) on disk.

### AWS S3 Support

synco on the server now automatically analyzes the `resource` publishing configuration of Neos Flow, and
if configured with [assets published to AWS S3](https://github.com/flownative/flow-aws-s3), it will detect
this and download the assets directly from S3 then.


### Smart Transfer by default

When analyzing file sizes of a typical large Neos/Flow installation, we found the following:

- for persistent resources (=files), it is typical that about 80% of a project's files are **auto-generated
  image thumbnails** in various sizes, which can be regenerated.
- database wise, the table `neos_neos_eventlog_domain_model_event` is often responsible for 80% of the database
  size, but it is rarely needed locally.

Thus, **in the default Smart Transfer mode**, the system does the following:

- we only include persistent resources in the `neos_flow_resourcemanagement_persistentresource` table
  and corresponding files which are **not thumbnails** - by cross-checking with the `neos_media_domain_model_thumbnail`
  table.
- We do not download the contents of `neos_media_domain_model_thumbnail`.
- we do not download the contents of `neos_neos_eventlog_domain_model_event`

To disable smart transfers (e.g. to debug an issue where you need *exactly the same as on the server*), you can run
`synco serve --all`:

```bash
curl https://sandstorm.github.io/synco/serve | sh -s - --all
```
