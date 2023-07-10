<h1 align="center">synco</h1>
<p align="center">an intelligent Database and File Dump Downloader for synchronizing content between production, staging, and local development</p>

<p align="center">

<a style="text-decoration: none" href="https://github.com/sandstorm/synco/releases">
<img src="https://img.shields.io/github/v/release/sandstorm/synco?style=flat-square" alt="Latest Release">
</a>

<a style="text-decoration: none" href="https://opensource.org/licenses/MIT">
<img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square" alt="License: MIT">
</a>

</p>

----

<p align="center">
<strong><a href="https://sandstorm.github.io/synco/#/installation">Installation</a></strong>
|
<strong><a href="https://sandstorm.github.io/synco/#/docs">Documentation</a></strong>
|
<strong><a href="https://sandstorm.github.io/synco/#/CONTRIBUTING">Contributing</a></strong>
</p>

----

<!-- TOC -->
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Usage Server-to-Server](#usage-server-to-server)
- [License](#license)
<!-- TOC -->
* [Architecture](https://sandstorm.github.io/synco/#/architecture)

In many web projects, we at some point needed to **synchronize content from production to local development**,
in order to reproduce and fix some bugs, test huge updates, or develop further features. We have implemented
lots of different synchronization scripts - but they were often kind of half-baked, and had some uncovered
edge cases.

Especially in bigger projects which f.e. have multiple gigabytes of persistent resources like images which must
be downloaded sometimes, we need the flexibility to choose whether we only need a database dump, or whether
we need all the files. And if possible, we only want to download changed files instead of all of them.

**This problem is what Synco solves.** Right now it is working **only for Neos/Flow projects**, though this will
change really soon (and is already prepared).

Synco is not the right fit if you want to run it inside some batch jobs like backups - I recommend to use
[other solutions like Borgbackup](https://www.borgbackup.org/) there.


# Features

```text
┌─────────────────────────────┐                      ┌─────────────────────────────┐
│         synco serve         │                      │        synco receive        │
│                             │                      │                             │
│      detect framework       │─────────────────────▶│ downloads and decrypt dump  │
│   produce encrypted dump    │                      │  (planned) import to local  │
│                             │                      │          instance           │
└─────────────────────────────┘                      └─────────────────────────────┘
   on your production server                             on your local instance     
```

Features:

* **Portable** written in Golang with no external dependencies. Install by downloading a single binary.
* **Transfer** files and database dumps from your production system
* **Re-use** the existing HTTP server which normally exists in a web-project (by placing the dumps in the public web folder
  of the project)
* **encrypts all dumps**; so nothing is transferred unencrypted. No unencrypted files temporary files are written.
* **Auto-Detects frameworks**, so it knows how to connect to the database. Supported right now:
  * **Neos / Flow Applications**
    * with local Resources
    * **NEW [since 1.2.0](./whats-new.md): with resources stored in S3**
    * **NEW [since 1.2.0](./whats-new.md): Smart Transfer - not downloading thumbnail resources for up to 80% size reduction.** 
  * (later, other frameworks will be added here)
* **multiple file-sets** supported. This means you can choose to only sync your database, but not your binary resources/assets.
* **Speed Optimized**: publicly available binary assets are not zipped extra; but the already-public files are simply downloaded.
  Resources which already exist locally and have the same file size and modification date are never re-downloaded.
* **no extra SQL client needed**: We package a custom implementation of `mysqldump` into the binary.
  * currently supported databases:
    * **MySQL**
    * (Postgres support planned)
* **auto-cleanup**: remove dumps when tool is stopped

# Installation

Run the following command on your developer machine, and this will download `synco` and place it on your `PATH`:

**macOS**
```bash
brew install sandstorm/tap/synco
```

# Usage

On your production server, run the following command **in the work directory of your application**:

```sh
curl https://sandstorm.github.io/synco/serve | sh -s -

# for verbose mode, run with "--debug" at the end.
curl https://sandstorm.github.io/synco/serve | sh -s - --debug

# for downloading ALL files and ALL database tables (including things
# like thumbnails which can be regenerated on the client)
curl https://sandstorm.github.io/synco/serve | sh -s - --all


# For Neos/Flow: To use a specific Flow context, do the following:
export FLOW_CONTEXT=Production
curl https://sandstorm.github.io/synco/serve | sh -s -
```

This will (on the prod server):

- Download `synco-lite` (in the version needed for your environment) and start it.
- Detect which framework is in use
- dump and encrypt the database by using the DB credentials of your application
- build an encrypted index of all public binary Resources of your system
- Show you a CLI call like `synco download [token] [password]`.

To download the dump, **on your local machine**, you run the CLI call printed out; and follow the wizard:
- You need to specify the host server (as synco cannot know under what URL the production system is reachable).
- You can choose what file-sets to download.

> **What is the difference between synco and synco-lite?**
>
> `synco-lite` is a binary-size-optimized package of Synco which only contains the code to run `synco serve`.
> This makes Synco more quick to run on the server side, where the tool is downloaded at first use.
>
> `synco` is the tool which contains all features, but comes with a bigger package size.

# Usage Server-to-Server

On the first host (where you want to download from), run the synco command as usual (see above).

On the second host, where you want to download to, you can run synco without installing it like this:

```sh
curl https://sandstorm.github.io/synco/synco | sh -s - <your-synco-arguments-as-shown-from-the-other-command-output>
# for example
curl https://sandstorm.github.io/synco/synco | sh -s - receive ts-...... ..........
```

# License

This project is licensed under the MIT license.
