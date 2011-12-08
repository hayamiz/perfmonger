 PerfMonger
============

PerfMonger is an yet anothor performance monitoring tool.


 Prerequisites
---------------

  * GLib 2
  * sysstat


 How to build
--------------

    $ ./configure
    $ make install


 How to use
------------

### Monitor IO performance of /dev/sda for each 0.1 second

    $ pgr -i 0.1 -d sda

