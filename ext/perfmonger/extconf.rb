#!/usr/bin/env ruby
# coding: utf-8

require 'mkmf'

$objs = ["perfmonger_record.o", "common.o", "ioconf.o", "iostat.o", "mpstat.o", "rd_stats.o"]

$cleanfiles += ["perfmonger_record.o", "common.o", "ioconf.o", "iostat.o", "mpstat.o", "rd_stats.o"]

create_makefile 'perfmonger/perfmonger_record'

mk = open('Makefile').read

mk.gsub!(/^LDSHARED = .*$/, "LDSHARED = $(CC)")
mk.gsub!(/^DLLIB = .*$/, "DLLIB = $(TARGET)")

open('Makefile', 'w') do |f|
  f.write(mk)
end
