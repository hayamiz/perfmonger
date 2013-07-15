
AM_CFLAGS = -g -Wall

if WITH_CUTTER
TESTS = run-test.sh
TESTS_ENVIRONMENT = NO_MAKE=yes CUTTER="$(CUTTER)"

noinst_LTLIBRARIES = test-perfmonger.la
endif

LDFLAGS += -module -rpath $(libdir) -avoid-version -no-undefined $(GLIB_LIBS)

LIBS += $(CUTTER_LIBS) $(GCUTTER_LIBS)

INCLUDES =					\
	 $(CUTTER_CFLAGS)			\
	 $(GCUTTER_CFLAGS)			\
	 $(GLIB_CFLAGS)				\
	-I$(top_srcdir)/src

test_perfmonger_la_SOURCES = test-perfmonger.c test.h
test_perfmonger_la_LIBADD = $(top_builddir)/src/libperfmonger.la

echo-cutter:
	@echo $(CUTTER)
