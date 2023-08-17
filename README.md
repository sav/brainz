# Brainz

A simple script to search and delete [ListenBrainz](https://listenbrainz.org) listens.

## Installation

```
go install github.com/sav/brainz@latest
```

## Usage

First define `BRAINZ_TOKEN` environment with an API Token from [ListenBrainz](https://listenbrainz.org) website:

```
export BRAINZ_TOKEN=<token>
```

### Searching

```
./brainz -l -u <user> -s <regexp>
```

### Deleting

```
./brainz -d -u <user> -s <regexp>
```
