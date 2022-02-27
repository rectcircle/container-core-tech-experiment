// gcc src/c/01-namespace/01-mount/pivot_root/main.c && sudo ./a.out

// 本例参考了：https://man7.org/linux/man-pages/man2/pivot_root.2.html#EXAMPLES

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
    "export PATH=/bin && ls / && ls /bin",
    NULL};

char *const new_root = "data/busybox/rootfs";
char *const put_old = "data/busybox/rootfs/.oldrootfs";
char *const put_old_on_new_rootfs = "/.oldrootfs";

int new_namespace_func(void *args)
{
    // 首先，需要阻止挂载事件传播到其他 Mount Namespace，参见：https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#NOTES
    // 如果不执行这个语句， cat /proc/self/mountinfo 所有行将会包含 shared，这样在这个子进程中执行 mount 其他进程也会受影响
    // 关于 Shared subtrees 更多参见：
    //   https://segmentfault.com/a/1190000006899213
    //   https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#SHARED_SUBTREES
    // 下面语句的含义是：重新递归挂（MS_REC）载 / ，并设置为不共享（MS_SLAVE 或 MS_PRIVATE）
    // 说明：
    //   MS_SLAVE 换成 MS_PRIVATE 也能达到同样的效果
    //   等价于执行：mount --make-rslave / 命令
    if (mount(NULL, "/", NULL, MS_SLAVE | MS_REC, NULL) == -1)
        errExit("mount-MS_SLAVE");
    // 确保 new_root 是一个挂载点
    if (mount(new_root, new_root, NULL, MS_BIND, NULL) == -1)
        errExit("mount-MS_BIND");
    // 切换根挂载目录，将 new_root 挂载到根目录，将旧的根目录挂载到 put_old 目录下
    // - new_root 和 put_old 必须是一个目录
    // - new_root 和 put_old 不能和当前根目录相同。
    // - put_old 必须是 new_root 的子孙目录
    // - new_root 必须是挂载点的路径，但不能是根目录。如果不是的话，可以通过 mount bind 方式转换为一个挂载点（参见上一个命令）。
    // - 旧的根目录必须是挂载点。
    // 更多参见：https: // man7.org/linux/man-pages/man2/pivot_root.2.html
    // 此外，可以通过 pivot_root(".", ".") 来实现免除创建临时目录，参见： https://github.com/opencontainers/runc/commit/f8e6b5af5e120ab7599885bd13a932d970ccc748
    if (pivot_root(new_root, put_old) == -1)
        errExit("pivot_root");
    // 根目录已经切换了，所以之前的工作目录已经不存在了，所以需要将 working directory 切换到根目录
    if (chdir("/") == -1)
        errExit("chdir");
    // 取消挂载旧的根目录路径
    if (umount2(put_old_on_new_rootfs, MNT_DETACH) == -1)
        perror("umount2");
    printf("=== new mount namespace and pivot_root process ===\n");
    execv(child_args[0], child_args);
    errExit("execv");
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
    // clone 库函数声明为：
    // int clone(int (*fn)(void *), void *stack, int flags, void *arg, ...
    // 		  /* pid_t *parent_tid, void *tls, pid_t *child_tid */);
    // 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
    pid_t p1 = clone(new_namespace_func, child_stack + STACK_SIZE, SIGCHLD | CLONE_NEWNS, NULL);
    if (p1 == -1)
        errExit("clone");
    waitpid(p1, NULL, 0);
    return 0;
}
