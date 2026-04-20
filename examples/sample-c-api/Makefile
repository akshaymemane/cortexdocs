CC      = cc
CFLAGS  = -Wall -O2 -DMG_ENABLE_LINES=1
TARGET  = server

MONGOOSE_TAG  = 7.14
MONGOOSE_BASE = https://raw.githubusercontent.com/cesanta/mongoose/$(MONGOOSE_TAG)

.PHONY: all run clean

all: $(TARGET)

mongoose.h:
	@echo "Fetching mongoose $(MONGOOSE_TAG)..."
	curl -fsSL $(MONGOOSE_BASE)/mongoose.h -o mongoose.h

mongoose.c: mongoose.h
	curl -fsSL $(MONGOOSE_BASE)/mongoose.c -o mongoose.c

$(TARGET): server.c mongoose.c mongoose.h users.h
	$(CC) $(CFLAGS) -o $(TARGET) server.c mongoose.c

run: $(TARGET)
	./$(TARGET)

clean:
	rm -f $(TARGET) mongoose.h mongoose.c
