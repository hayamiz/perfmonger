
AM_CFLAGS = -g -Wall
TESTS = run-test.sh
TESTS_ENVIRONMENT = NO_MAKE=yes CUTTER="$(CUTTER)"

LDFLAGS += -module -rpath $(libdir) -avoid-version -no-undefined $(GLIB_LIBS)

LIBS += $(CUTTER_LIBS) $(GCUTTER_LIBS)

INCLUDES =					\
	 $(CUTTER_CFLAGS)			\
	 $(GCUTTER_CFLAGS)			\
	 $(GLIB_CFLAGS)				\
	-I$(top_srcdir)/src

noinst_LTLIBRARIES = test-perfmonger.la

test_perfmonger_la_SOURCES = test-perfmonger.c
test_perfmonger_la_LIBADD = $(top_builddir)/src/libperfmonger.la


echo-cutter:
	@echo $(CUTTER)
