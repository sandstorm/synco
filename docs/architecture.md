# Architecture of Synco

## The Problem We want to Solve

**Working locally and on staging with as-live-as-possible content**

In our projects, we often have the need to work with production data locally, to debug a certain issue, or further develop new features.

In our agency, we use lots of different technologies and infrastructures. Thus, we want a solution with the following properties:

* **Framework Agnostic:** The approach must work with different programming frameworks and even different programming languages. It should, however, have defaults out of the box to work with certain frameworks.
* **Infrastructure Agnostic:** Sometimes, we host staging or prod instances in Kubernetes. Sometimes on Docker. Sometimes on Bare Metal. Most often it's Linux, but sometimes it's running on BSD as well.
* **Self Contained:** We do not want to make any assumptions about available programs on the host.
* **Easy to learn and use:** It should ideally be plug and play.
* **As Little Data Transfer As Possible**: Because we sometimes have big content dumps which we need to transfer, we want to transfer as little data as possible. This is not so relevant between servers, but is hugely relevants for development instances (because there, Bandwidth is way more limited).
* **Secure over the wire**: We transfer the data over the internet; thus we don't want to trust people there. We need to encrypt all data in-flight.

**Assumptions** we can take for granted when building our solution

* We assume that we have access to an interactive shell for the **source** instance where we want to copy from. This could be via SSH, via `kubectl` or via `docker exec` (or a combination thereof). We call this a **trusted control channel**.
* We optimize for the **interactive case**, and not for the batch machine-to-machine case.

**Prior Art**

* Go CLI
    * Wizard: https://github.com/pterm/pterm
    * Basic CLI API: https://cobra.dev/
* Hot Reload: https://github.com/cosmtrek/air
* Database:
    * https://github.com/JamesStewy/go-mysqldump
    * fork with postgres support: https://github.com/conneqtech/go-mysqldump (adapted to own needs)

## Solution Concept

We transfer always from source to destination:

* the source is usually located on the production server.
* The destination is usually the local system where you develop.

While the source is usually exposed to the internet (it is a server), the destination can be inside a private network.
This means all connections need to be initiated from the destination.

Conversely, the source server usually does NOT have synco installed; while on the target you might have it installed
(because you use it locally). Because we need to install synco on the source server in an ad-hoc manner, we create
an extra `synco-lite` binary which has a way smaller file size than the general-purpose synco tool.

For now we assume the source server is reachable via HTTP from the destination; and we can use this
for transferring files. In the future, we might have other ways of transfer. This means for confidentiality,
we need to encrypt all files before making them available on a public HTTP endpoint.

The destination server will only fetch data from the source; and never the other way round. It basically
"pulls the data down" from the source server.

## File transfer method

on the source server (e.g. the production system), the user needs to log in, and then invoke the `synco-lite` executable. This does:

* install synco-source via a shell script
* Detect which framework is used. E.g. for Flow/Neos or Symfony, synco then knows how to create a database dump (e.g. for MySQL, using go-mysqldump or
  pingcap dumpling; and for Postgres some pgx based solution??)
* Publish a metadata file and encrypt it which shows the current status.
* Create the database dump and encrypt it.
* Create a file mapping for data/persistent in flow - as we do not need to re-compress static assets which are available online.
* Generate the target synco command which contains the private key, the server, and the sync session.

The user then on the destination (usually his local machine) runs synco download, as shown by the wizard above, with all arguments.

* then the destination synco client downloads the metadata (waiting for ready state if needed); downloads the files
  (only if local files are not existing / older); installs them into the local instance.
* At the end a message is printed that the synco-source instance should be terminated.

On the server, when terminating synco-source (kill hook), we remove all published web files.

Streamlined workflow (not yet implemented)

Based on the workflow above, we can implement an even more streamlined “synco” workflow which is run on the destination
(=the local machine); which connects to the source via some out of band mechanism like kubectl or SSH; and orchestrates
the process above.

## Sync Format

Goals:
- Incremental Syncing (Don't download files you already have)
- Encrypted / non Encrypted Syncing (depending if the source files are already public or not)
- Partial Syncing (only download what you want (no resources if you do not want them))

```
meta.json
{
  state: "Created|Initializing|Ready",
  frameworkName: "Neos"
  files: [
    {
      name: "db-dump",
      type: "single",
      single: {
        fileName: "db-dump.sql.enc"
        sizeBytes: 500000
      }
    },
    {
      name: "persistent",
      type: "publicFiles",
      publicFiles: {
        indexFileName: "persistent-index.json.enc",
        sizeBytes: 500000
      }
    }
  ]
}

persistent-index.json:

{
  "Foo/Bar/bla": {
    "mtime": 123456789,
    "sizeBytes": 500000,
    "publicUri": "<BASE>/_Web/Resources/....." 
  }
}
```

## Previous Idea Iterations

### transfer via WebRTC

* https://github.com/Antonito/gfile - gfile is a WebRTC based file exchange software.
* Too complex.

### Transfer via Tailscale

* Embedded Tailscale
    * https://github.com/tailscale/tailscale/blob/v1.32.0/tsnet/example/tshello/tshello.go
* needs infrastructure

## ideas

* three-way (control server !== target)
