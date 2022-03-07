// gcc src/c/01-namespace/scenes/01-join_to_ns.c && sudo ./a.out

// 参考：https://man7.org/linux/man-pages/man2/setns.2.html#EXAMPLES
#define _GNU_SOURCE    // Required for enabling clone(2)
#include <sys/wait.h>  // For waitpid(2)
#include <sys/mount.h> // For mount(2)
#include <sys/mman.h>  // For mmap(2)
#include <sched.h>     // For clone(2)
#include <signal.h>    // For SIGCHLD constant
#include <stdio.h>     // For perror(3), printf(3), perror(3)
#include <unistd.h>    // For execv(3), sleep(3)
#include <stdlib.h>    // For exit(3), system(3)
#include <limits.h>    // For PATH_MAX
#include <fcntl.h>     // For O_RDONLY, O_CLOEXEC
#include <sys/syscall.h> // For  SYS_* constants

#define errExit(msg)    do { perror(msg); exit(EXIT_FAILURE); \
                               } while (0)

static int
pivot_root(const char *new_root, const char *put_old)
{
    return syscall(SYS_pivot_root, new_root, put_old);
}

#define STACK_SIZE (1024 * 1024)

char *const child_args[] = {
    "/bin/sh",
    "-xc",
    "export PATH=/bin:$PATH && ls /",
    NULL};

char *const put_old = "data/busybox/rootfs2/.oldrootfs";
char *const new_root = "data/busybox/rootfs2";
char *const new_root_old_proc = "data/busybox/rootfs2/.oldproc";
char *const put_old_on_new_rootfs = "/.oldrootfs";

pid_t fork_proccess(char *const *argv)
{
    pid_t p = fork();
    if (p == 0)
    {
        execv(argv[0], argv);
        perror("exec");
        exit(EXIT_FAILURE);
    }
    return p;
}

int new_namespace_func(void *args)
{
    // mount 的传播性
    if (mount(NULL, "/", NULL, MS_PRIVATE | MS_REC, NULL) == -1)
        errExit("mount-MS_PRIVATE");
    // 确保 new_root 是一个挂载点
    if (mount(new_root, new_root, NULL, MS_BIND, NULL) == -1)
        errExit("mount-MS_BIND");
    // 绑定旧的 /proc
    if (mount("/proc", new_root_old_proc, NULL, MS_BIND, NULL) == -1)
        errExit("mount-MS_BIND-PROC");
    // 切换根挂载目录，将 new_root 挂载到根目录，将旧的根目录挂载到 put_old 目录下
    if (pivot_root(new_root, put_old) == -1)
        errExit("pivot_root");
    // 根目录已经切换了，所以之前的工作目录已经不存在了，所以需要将 working directory 切换到根目录
    if (chdir("/") == -1)
        errExit("chdir");
    // 取消挂载旧的根目录路径
    if (umount2(put_old_on_new_rootfs, MNT_DETACH) == -1)
        perror("umount2");
    printf("=== new mount namespace and pivot_root process ===\n");
    int pid = fork_proccess(child_args);
    if (pid == -1)
        perror("fork");
    waitpid(pid, NULL, 0);

    int fd = open("/.oldproc/1/ns/mnt", O_RDONLY | O_CLOEXEC);
    if (fd == -1)
        errExit("open");
    if (setns(fd, 0) == -1)
        errExit("setns");
    close(fd);
    printf("=== old mount namespace process by setns ===\n");
    pid = fork_proccess(child_args);
    if (pid == -1)
        perror("fork");
    waitpid(pid, NULL, 0);
}

int main()
{
    // 为子进程提供申请函数栈
    void *child_stack = mmap(NULL, STACK_SIZE,
                             PROT_READ | PROT_WRITE,
                             MAP_PRIVATE | MAP_ANONYMOUS | MAP_STACK,
                             -1, 0);
    if (child_stack == MAP_FAILED)
        errExit("mmap");
    // 创建新进程，并为该进程创建一个 Mount Namespace（CLONE_NEWNS），并执行 new_namespace_func 函数
    pid_t p1 = clone(new_namespace_func, child_stack + STACK_SIZE, SIGCHLD | CLONE_NEWNS | CLONE_NEWPID, NULL);
    if (p1 == -1)
        errExit("clone");
    waitpid(p1, NULL, 0);
    return 0;
}
