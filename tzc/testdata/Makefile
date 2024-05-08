# Find all .tzdata files in the current directory
SOURCES := $(wildcard *.tzdata)

# Generate target names by stripping the .tzdata extension and appending _tzif
TARGETS := $(SOURCES:%.tzdata=%_tzif)

# Default target
all: $(TARGETS)

# Rule to process each file
$(TARGETS): %_tzif : %.tzdata
	rm -rf $@
	zic $< -b fat -d $@

# Clean target to remove the directories
clean:
	rm -rf *_tzif

# Phony targets to avoid conflicts with existing directories
.PHONY: all clean
