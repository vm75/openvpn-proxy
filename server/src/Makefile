PROG = openvpn-proxy

CXXFLAGS += -W -Wall -DMG_ENABLE_LINES=1 -DMG_ENABLE_OPENSSL=1 -lssl -lcrypto --std=c++20 -lstdc++ -g

all: $(PROG)
# $(DEBUGGER) ./$(PROG) $(ARGS)

$(PROG): main.cpp
	$(CXX) mongoose.c -I.:/usr/include $(CXXFLAGS) $(EFLAGS) -o $(PROG) main.cpp

clean:
	rm -rf $(PROG) *.o *.dSYM *.gcov *.gcno *.gcda *.obj *.exe *.ilk *.pdb
