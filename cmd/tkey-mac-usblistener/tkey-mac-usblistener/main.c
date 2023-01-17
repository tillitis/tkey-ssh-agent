//
//  main.c
//  tkey-mac-usblistener
//
//  Created by Johan Carlberg on 2023-01-02.
//

#include <stdio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/usb/IOUSBLib.h>
#include <mach/mach.h>
#include <libproc.h>

static int                      gNumberOfProcessNames;
static const char **            gProcessNames;

static IONotificationPortRef    gNotifyPort;
static io_iterator_t            gRawAddedIter;
static io_iterator_t            gRawRemovedIter;

void signal_processes(int signal) {
    pid_t pids[2048];
    int bytes = proc_listpids(PROC_ALL_PIDS, 0, pids, sizeof(pids));
    int n_proc = bytes / sizeof(pids[0]);

    for (int i = 0; i < n_proc; i++) {
        struct proc_bsdinfo proc;
        int st = proc_pidinfo(pids[i], PROC_PIDTBSDINFO, 0, &proc, PROC_PIDTBSDINFO_SIZE);
        if (st == PROC_PIDTBSDINFO_SIZE) {
            for (int idx = 0; idx < gNumberOfProcessNames; idx += 1) {
                if (strcmp(gProcessNames[idx], proc.pbi_name) == 0) {
                    kill (pids[i], signal);
                }
            }
        }
    }
}

void NotifyProcesses(void) {
    signal_processes(SIGHUP);
}

void RawDeviceAdded(void *refCon, io_iterator_t iterator) {
    io_service_t usbDevice;

    while ((usbDevice = IOIteratorNext(iterator)) != IO_OBJECT_NULL) {
        printf ("USB device added\n");
        IOObjectRelease(usbDevice);
        NotifyProcesses();
    }
}

void RawDeviceRemoved(void *refCon, io_iterator_t iterator) {
    io_service_t usbDevice;

    while ((usbDevice = IOIteratorNext(iterator)) != IO_OBJECT_NULL) {
        printf ("USB device removed\n");
        IOObjectRelease(usbDevice);
        NotifyProcesses();
    }
}

int main (int argc, const char *argv[]) {
    mach_port_t             mainPort;
    CFMutableDictionaryRef  matchingDict;
    CFRunLoopSourceRef      runLoopSource;
    kern_return_t           kr;

    if (argc <= 1) {
        fprintf(stderr, "ERR: No process name arguments\n");
        return -1;
    }
    gNumberOfProcessNames = argc - 1;
    gProcessNames = argv + 1;

    //Create a master port for communication with the I/O Kit
    kr = IOMainPort(MACH_PORT_NULL, &mainPort);
    if (kr || !mainPort) {
        fprintf(stderr, "ERR: Couldn’t create a main I/O Kit port (%08x)\n", kr);
        return -1;
    }

    //Set up matching dictionary for class IOUSBDevice and its subclasses
    matchingDict = IOServiceMatching(kIOUSBDeviceClassName);
    if (!matchingDict) {
        fprintf(stderr, "Couldn’t create a USB matching dictionary\n");
        mach_port_deallocate(mach_task_self(), mainPort);
        return -1;
    }
  
    //To set up asynchronous notifications, create a notification port and
    //add its run loop event source to the program’s run loop
    gNotifyPort = IONotificationPortCreate(mainPort);
    runLoopSource = IONotificationPortGetRunLoopSource(gNotifyPort);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), runLoopSource, kCFRunLoopDefaultMode);

    //Retain additional dictionary references because each call to
    //IOServiceAddMatchingNotification consumes one reference
    matchingDict = (CFMutableDictionaryRef) CFRetain(matchingDict);
    matchingDict = (CFMutableDictionaryRef) CFRetain(matchingDict);
    matchingDict = (CFMutableDictionaryRef) CFRetain(matchingDict);
 
    //Now set up two notifications: one to be called when a raw device
    //is first matched by the I/O Kit and another to be called when the
    //device is terminated
    //Notification of first match:
    kr = IOServiceAddMatchingNotification(gNotifyPort, kIOFirstMatchNotification, matchingDict, RawDeviceAdded, NULL, &gRawAddedIter);
    if (kr) {
        fprintf(stderr, "Couldn't create device added notification (%08x)\n", kr);
        return -1;
    }
    //Iterate over set of matching devices to access already-present devices
    //and to arm the notification
    RawDeviceAdded(NULL, gRawAddedIter);

    //Notification of termination:
    kr = IOServiceAddMatchingNotification(gNotifyPort, kIOTerminatedNotification, matchingDict, RawDeviceRemoved, NULL, &gRawRemovedIter);
    if (kr) {
        fprintf(stderr, "Couldn't create device removed notification (%08x)\n", kr);
        return -1;
    }
    //Iterate over set of matching devices to release each one and to
    //arm the notification
    RawDeviceRemoved(NULL, gRawRemovedIter);

    //Finished with master port
    kr = mach_port_deallocate(mach_task_self(), mainPort);
    if (kr) {
        fprintf(stderr, "Couldn't deallocate Mach port (%08x)\n", kr);
        return -1;
    }
    mainPort = 0;
 
    //Start the run loop so notifications will be received
    CFRunLoopRun();
 
    //Because the run loop will run forever until interrupted,
    //the program should never reach this point
    return 0;
}
