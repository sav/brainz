# Brainz

A simple script to search and delete [ListenBrainz](https://listenbrainz.org) entries.

## Installation
```
go install github.com/sav/brainz@latest
```

## Usage

First define `LISTENBRAINZ_TOKEN` environment with an API Token from [ListenBrainz](https://listenbrainz.org) website:
```
export LISTENBRAINZ_TOKEN=<token>
```
Then run:
```
brainz -h
```

### Searching

By regular expression (case insensitive):
```
brainz -u <user> -s <regexp>
```
By elapsed time:
```
brainz -u <user> -t <duration>
```
Where `duration` is number followed by **`m`** (minutes), **`h`** (hours), **`d`** (days) or **`y`** (years).

### Deleting

```
brainz -u <user> -s <regexp> -d
```
```
brainz -u <user> -t <duration> -d
```

### Examples

List all "Pink Floyd" listens:
```
brainz -u sav10sena -s "Pink\s*Floyd"
```
Remove all listens from the past 30 minutes.
```
brainz -u sav10sena -t 30m -d
```
