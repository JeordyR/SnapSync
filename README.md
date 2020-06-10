# SnapSync

This is a utility for running snapraid syncs and scrubs on a schedule with various filters and checks. Inspired by https://github.com/Chronial/snapraid-runner.

It is intended to be run as a cron job with something like : `0 4 * * * /path/to/snapsync -c /path/to/snapsync.yaml 2>&1 | /usr/bin/logger -t snapsync`

## How To Use
- Download [the latest release](https://github.com/JeordyR/SnapSync/releases) and extract the binary
- Place the binary in a directory within your path (/usr/bin for example) to execute without /path/to, or place anywhere
- Copy the [example configuration file](https://github.com/JeordyR/SnapSync/blob/master/snapsync.yaml) and make tweaks as needed
- Run the tool with `snapsync -config /path/to/config` or `/path/to/snapsync -config /path/to/config` depending on where you placed the binary

## Features
- Self-updating binary: On every execution the tool will check for a new release and update itself in-place
- Pushover notifications: Optional notifications to pushover when snapsync starts and with status updates along the way
- Runs diff before sync to see how many files were deleted with optional cutoff to prevent execution
- Optionally performs scrub after sync, with scheduling options to define how often scrub should run
- Optionally runs `snapraid touch` before sync to sort out timestamp issues
- Optionally runs `snapraid status`, parses the response, and logs/pushovers it (oldest block, percentage scrubbed, last scrub, any warning lines, etc.)

## Configuration
- Base Settings:
    - **Executable**: Path to the snapraid executable, usually `/usr/bin/snapraid`, run `which snapraid` to get yours
- Log Settings:
    - **LogFile**: Path to the logfile where snapsync should store logs
- Touch Settings:
    - **TouchEnabled**: Boolean (true/false) whether touch should be run before sync
- Threashold Settings:
    - **DeleteThreashold**: Maximum number of deleted files to allow before skipping sync operations (0 for unlimited)
- Scrub Settings:
    - **ScrubEnabled**: Boolean (true/false) whether scrub should be performed after sync or not
    - **ScrubPercentage**: The scrub percentage to pass to the snapraid scrub command (see below)
    - **ScrubOlderThan**: The older-than value to pass to the snapraid scrub command (see below)
    - **ScrubDaysOfWeek**: List of days of the week that scrub should be performed on
- Status Settings:
    - **OutputStatus**: Boolean (true/false) whether to run `snapraid status` and output the parsed response to log and pushover (if enabled)
- Pushover Settings:
    - **PushoverEnabled**: Boolean (true/false) whether pushover notifications should be sent or not
    - **PushoverAppKey**: Pushover app key for this application
    - **PushoverUserKey**: Pushover user key for your account

## Scrub Explanation

Scrub command description/explanation pulled from the [official docs](https://www.snapraid.it/manual)

Scrub Definition:
> Scrubs the array, checking for silent or input/output errors in data and parity disks.

Amount scrubbed by default:
>For each command invocation, about the 8% of the array is checked, but nothing that was already scrubbed in the last 10 days. This means that scrubbing once a week, every bit of data is checked at least one time every three months.

Percentage:
>Scrub the exact percentage of blocks.

Older-Than Option:
>older-than option to define how old the block should be. The oldest blocks are scrubbed first ensuring an optimal check.

For example, with the current defaults in the example config file, scrub is run every day. 10% of the array is scrubbed each day, and only scrubs blocks that have not been scrubbed in the last 10 days. So every bit of data is checked at least once every 10 days.

## Pushover Example:

Bellow is an example of the messages received on pushover from a recent run of snapsync with settings matching the default example config (with pushover enabled):

![pushover example](https://github.com/JeordyR/SnapSync/blob/master/.github/images/pushover-example.jpg?raw=true)
