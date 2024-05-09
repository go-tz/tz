# Find all .tzdata files in the current directory
SOURCES := $(wildcard *.tzdata)

# Generate target names by stripping the .tzdata extension and appending _tzif
TARGETS := $(SOURCES:%.tzdata=generated_tzif/%)

# Default target
all: $(TARGETS)

# Rule to process each file
$(TARGETS): generated_tzif/% : %.tzdata
	zic $< -d $@

# Clean target to remove the directories
clean:
	rm -rf generated_tzif/*

# Phony targets to avoid conflicts with existing directories
.PHONY: all clean
