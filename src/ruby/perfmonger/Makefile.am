include ../../../common.mk

SUBDIRS = command

EXTRA_DIST = cli.rb config.rb.in version.rb.in

perfmonger_rubylib_perfmongerdir = $(perfmonger_rubylibdir)/perfmonger

dist_perfmonger_rubylib_perfmonger_DATA = cli.rb config.rb version.rb

config.rb: config.rb.in Makefile
	$(edit) $< > $@.tmp
	chmod +x $@.tmp
	mv $@.tmp $@

