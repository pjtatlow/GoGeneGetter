#!/usr/bin/env python2

import sys
import subprocess
import psutil
import time
import threading
import signal
import os

class Profiler:
    """ profiles a server based on a number of functions """
    def __init__(self, funcList, serverPath, nTList=[1,10,100]):
        print "Starting Iris"
        self.subproc = subprocess.Popen(["go","run",serverPath], cwd=serverPath[:serverPath.rfind("/")], preexec_fn=os.setsid)
        time.sleep(4)
        self.proc = psutil.Process(self.subproc.pid)
        self.nTList = nTList # Number of threads
        self.funcList = funcList
        self.data = {}
        
    def profile(self):
        for nT in self.nTList:
            self.data[nT] = {}
            for func in self.funcList:
                threads = []
                self.data[nT][func.__name__] = {}
                profThread = ProfilingThread(self.proc, self.data[nT][func.__name__])
                profThread.start()
                for i in xrange(nT):
                    t = Tester(func)
                    t.start()
                    threads.append(t)
                for t in threads:
                    t.join()
                profThread.stop()
                profThread.join()
        return self.data
                
class Tester(threading.Thread):
    """ A thread that runs a function and can modify data """
    def __init__(self, func, data=None):
        super(Tester, self).__init__()
        self.data = data
        self.func = func
        
    def run(self):
        self.func() # Won't actually do anything with the data object as of now...
                

class ProfilingThread(threading.Thread):
    """ A stoppable thread that tracks a process's time, cpu, and memory usage """
    def __init__(self, proc, data):
        super(ProfilingThread, self).__init__()
        self._stop = threading.Event()
        self.data = data
        self.proc = proc
        
    def stop(self):
        self._stop.set()
    
    def stopped(self):
        return self._stop.isSet()
    
    def run(self):
        cpu = []
        mem = []
        self.tStart = time.time()
        while not self.stopped():
            mem.append(self.proc.memory_percent())
            cpu.append(self.proc.cpu_percent(interval=0.1))
        self.tStop = time.time()
        self.data.update({'time': self.tStop - self.tStart,
               'max_cpu': max(cpu),
               'avg_cpu': sum(cpu)/float(len(cpu)),
               'max_mem': max(mem),
               'avg_mem': sum(mem)/float(len(mem))})
    
    

if __name__ == "__main__":
    serverPath = sys.argv[1] # really should use argparse
    
    def sleep1Sec():
        time.sleep(1)
    
    def sleepHalfSec():
        time.sleep(.5)
    
    funcList = [sleep1Sec, sleepHalfSec]
    
    p = Profiler(funcList, serverPath)
    try:
        print p.profile()
#    except:
#        print "Some error occurred.  Exiting."
#        sys.exit()
    finally:
        print "Killing Iris..."
        os.killpg(os.getpgid(p.subproc.pid), signal.SIGTERM)