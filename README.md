 PerfMonger
============

[![Build Status](https://travis-ci.org/hayamiz/perfmonger.svg?branch=master)](https://travis-ci.org/hayamiz/perfmonger)

[![wercker status](https://app.wercker.com/status/44c3ade6a2406d337df6d93097a52fdf/m "wercker status")](https://app.wercker.com/project/bykey/44c3ade6a2406d337df6d93097a52fdf)

PerfMonger is a system performance monitor which enables high-resolution and holistic performance measurement with the programmer friendly interface.

* High-resolution: sub-second level monitoring is possible!
* Holistic performance measurement: monitoring CPU, Disk I/O, Network all at once.
* Programmer friendly: PerfMonger speaks monitoring results in JSON format, which makes later performance analysis much easier (ex. [jq](https://github.com/stedolan/jq)).

**CAUTION: PerfMonger is still in early stage, so there may be a drastic change in the future. Do not use it for critical jobs**

## Target platform

  * GNU/Linux
  * Mac OS X (experimental support)

## How to installation

    gem install perfmonger

You need gnuplot 4.6.0 or later build with cairo terminals for plotting measurement data with `perfmonger plot` command.

### Build from source

You need Ruby 1.9.3 or later, and Go 1.8 or later to build perfmonger.

    bundle
    rake build

## How to use: case study

### Monitor IO performance of /dev/sda for each 0.1 second

    $ perfmonger record -i 0.1 -d sda

### Monitor CPU usage for each 0.1 second

    $ perfmonger record -i 0.1

### Monitor CPU usage and IO performance of /dev/sda, sdb for each 0.1 second

    $ perfmonger record -i 0.1 -d sda -d sdb
