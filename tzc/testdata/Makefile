# Find all .tzdata files in the current directory
TZDATA := $(shell find tzdata -type f -name "*.tzdata")

# Extract the case name from the file name. For example, tzdata/initial_time/sample.tzdata -> initial_time/sample.
CASES := $(TZDATA:tzdata/%.tzdata=%)

# Create the target file names. For example, initial_time/sample -> tzif/initial_time/sample.
TARGETS := $(CASES:%=tzif/%)

# Default target
all: $(TARGETS)

# Rule to process each file
$(TARGETS): tzif/% : tzdata/%.tzdata
	zic -v $< -d $@

# Clean target to remove the directories
clean:
	rm -rf tzif/*

# Phony targets to avoid conflicts with existing directories
.PHONY: all clean
