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

You need Ruby 2.2 or later, and Go 1.8 or later to build perfmonger.

    bundle
    rake build

## Getting started

Basic usage of PerfMonger is:

* Run `perfmonger record` to record performance information logs
* Run `perfmonger play` to show performance information logs in JSON format

`perfmonger play` repatedly prints records of system performance including CPU
usages, disk usages, and network usages. One line includes only one record, so
you can easily process output records with `jq`, or any scripting languages like
Ruby, Python, and etc.

Pretty-printed structure of a record is as follows:

```
{"time": 1500043743.504, # timestamp in unix epoch time
  "cpu": {               # CPU usages
    "num_core": 2,         # the number of cores
    "all": {               # aggregated CPU usage (max = num_core * 100%)
      "usr": 50.0,
      "sys": 50.0,
      "idle": 100.0,
      ...
    },
    "cores": [             # Usage of each CPU core
      {
        "usr": 25.0,
        "sys": 25.0,
        "idle": 50.0,
        ...
      },
      {
        "usr": 25.0,
        "sys": 25.0,
        "idle": 50.0,
        ...
      }
    ]
  },
  "disk": {             # Disk usages
    "devices": ["sda"],   # List of disk devices
    "sda": {              # Usage of device 'sda'
      "riops": 10.0,        # The number of read I/O per second
      "wiops": 20.0,        # The number of write I/O per second
      "rkbyteps": 80.0,     # Read transfer rate in KB/s
      "wkbyteps": 160.0,    # Write transfer rate in KB/s
      ...
    }
    "total": {           # Aggregated usage of all devices
      "riops": 10.0,
      "wiops": 20.0,
      "rkbyteps": 80.0,
      "wkbyteps": 160.0,
      ...
    }
  }
}
```


## Typical use cases

### `perfmonger live`: live performance monitoring

    $ perfmonger live

### Monitor IO performance of /dev/sda for each 0.1 second

    $ perfmonger record -i 0.1 -d sda

### Monitor CPU usage for each 0.1 second

    $ perfmonger record -i 0.1

### Monitor CPU usage and IO performance of /dev/sda, sdb for each 0.1 second

    $ perfmonger record -i 0.1 -d sda -d sdb
