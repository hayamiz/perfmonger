 PerfMonger
============

PerfMonger is an yet anothor performance monitoring tool.

** PerfMonger is still in early stage, so there may be a drastic
   change in the future. Do not use it for critical jobs **

 Prerequisites
---------------

  * GLib 2
  * sysstat


 How to build
--------------

    $ ./configure
    $ make install


 How to use: case study
------------------------

### Monitor IO performance of /dev/sda for each 0.1 second

    $ perfmonger -i 0.1 -d sda

### Monitor CPU usage for each 0.1 second

    $ perfmonger -i 0.1

### Monitor CPU usage and IO performance of /dev/sda, sdb for each 0.1 second

    $ perfmonger -i 0.1 -d sda -d sdb
