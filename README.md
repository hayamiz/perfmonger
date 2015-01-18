 PerfMonger
============

PerfMonger is an yet anothor performance measurement/monitoring tool
speaking JSON.

**CAUTION: PerfMonger is still in early stage, so there may be a drastic change in the future. Do not use it for critical jobs**

## Target platform

  * GNU/Linux

## Prerequisites

  * Ruby 1.9.3 or later
  * gnuplot 4.6.0 or later (optional)

Note: You need Cutter unit testing framework for building/running tests.

## How to install

    gem install perfmonger

### Build from source

    rake build

## How to use: case study

### Monitor IO performance of /dev/sda for each 0.1 second

    $ perfmonger record -i 0.1 -d sda

### Monitor CPU usage for each 0.1 second

    $ perfmonger record -i 0.1

### Monitor CPU usage and IO performance of /dev/sda, sdb for each 0.1 second

    $ perfmonger record -i 0.1 -d sda -d sdb

### Plot CPU and IOPS

    $ perfmonger record -i 0.1 -C -d sda > /tmp/perfmonger.log & sleep 10; pkill perfmonger
    $ perfmonger plot -o /path/to/output_dir/ -Tpng /tmp/perfmonger.log
    $ display /path/to/output_dir/read-iops.png
    $ display /path/to/output_dir/cpu.png

![Sample image of IOPS graph](https://raw.github.com/hayamiz/perfmonger/master/misc/sample-read-iops.png)
![Sample image of CPU usage graph](https://raw.github.com/hayamiz/perfmonger/master/misc/sample-cpu.png)

## Special Thanks

Large portion of PerfMonger comes from
[SYSSTAT](http://sebastien.godard.pagesperso-orange.fr/) codebase. Thanks for
their great work.
