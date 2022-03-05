// gcc src/c/01-namespace/04-pid/main.c && sudo ./a.out

#define _GNU_SOURCE	   // Required for enabling clone(2)
#include <sys/wait.h>  // For waitpid(2)
#include <sys/mount.h> // For mount(2)
#include <sys/mman.h>  // For mmap(2)
#include <sys/syscall.h> // For SYS_pidfd_open

#include <time.h>	   // For nanosleep(2)
#include <sched.h>	   // For clone(2)
#include <signal.h>	   // For SIGCHLD
#include <stdio.h>	   // For perror(3), printf(3), perror(3)
#include <unistd.h>    // For execv(3), sleep(3)
#include <stdlib.h>    // For exit(3), system(3)


#define errExit(msg)    do { perror(msg); exit(EXIT_FAILURE); \
                               } while (0)

#define STACK_SIZE (1024 * 1024)

void my_sleep(int sec)
{
	struct timespec t = {
		.tv_sec = sec,
		.tv_nsec = 0};
	// sleep 会被信号打断，因此通过 nanosleep 重新实现一下
	// https: // man7.org/linux/man-pages/man2/nanosleep.2.html
	while (nanosleep(&t, &t) != 0)
		;
}

// 进程 a：当前 bash，最终为 sleep infinity
// 进程 d：nohup sleep infinity 孤儿进程在该 PID Namespace 中，其 ppid 为 1
char *const proccess_a_args[] = {
	"/bin/bash",
	"-xc",
	"bash -c 'nohup sleep infinity >/dev/null 2>&1 &' \
	&& echo $$ \
	&& ls /proc \
	&& ps -o pid,ppid,cmd \
	&& kill -9 1\
	&& ps -o pid,ppid,cmd \
	&& exec sleep infinity \
	",
	NULL};

// 进程 b： 在该 PID Namespace 中，构造一个孤儿进程，其 ppid 为 0，在父 PID Namespace 中 为 1
char *proccess_b_args[] = {
	"/bin/bash",
	"-c",
	"",
	NULL};

// 进程 c： sleep infinity 进程在该 PID Namespace 中，其 ppid 为 0，在父 PID Namespace 中 ppid 为 主进程
char *const proccess_c_args[] = {
	"/bin/bash",
	"-c",
	"exec sleep infinity",
	NULL};

// 进程 e：
char *const proccess_e_args[] = {
	"/bin/bash",
	"-xc",
	"ls /proc \
	&& ps -eo pid,ppid,cmd | grep sleep | grep -v grep  \
	&& kill -9 $(ps -eo pid,ppid | grep $PPID | awk '{print $1}' | sed -n '2p') \
	&& ps -eo pid,ppid,cmd | grep sleep | grep -v grep \
	",
	NULL};

int new_namespace_func(void *args)
{
	// seq: 0s

	// 首先，需要阻止挂载事件传播到其他 Mount Namespace，参见：https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#NOTES
	// 如果不执行这个语句， cat /proc/self/mountinfo 所有行将会包含 shared，这样在这个子进程中执行 mount 其他进程也会受影响
	// 关于 Shared subtrees 更多参见：
	//   https://segmentfault.com/a/1190000006899213
	//   https://man7.org/linux/man-pages/man7/mount_namespaces.7.html#SHARED_SUBTREES
	// 下面语句的含义是：重新递归挂（MS_REC）载 / ，并设置为不共享（MS_SLAVE 或 MS_PRIVATE）
	// 说明：
	//   MS_SLAVE 换成 MS_PRIVATE 也能达到同样的效果
	//   等价于执行：mount --make-rslave / 命令
	if (mount(NULL, "/", NULL , MS_SLAVE | MS_REC, NULL) == -1)
		errExit("mount-MS_SLAVE");
	// 挂载当前 PID Namespace 的 proc
	// 因为在新的 Mount Namespace 中执行，所有其他进程的目录树不受影响
	// 等价命令为：mount -t proc proc /proc
	// mount 函数声明为：
	//    int mount(const char *source, const char *target,
	//              const char *filesystemtype, unsigned long mountflags,
	//              const void *data);
	// 更多参见：https://man7.org/linux/man-pages/man2/mount.2.html
	if (mount("proc", "/proc", "proc", 0, NULL) == -1)
		errExit("mount-proc");
	my_sleep(3);
	// seq: 3s
	printf("=== new pid namespace process ===\n");
	execv(proccess_a_args[0], proccess_a_args);
	perror("exec");
	exit(EXIT_FAILURE);
}

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

void set_pid_namespace(pid_t pid) {
	int fd = syscall(SYS_pidfd_open, pid, 0);
	if (fd == -1)
		errExit("pidfd_open");
	if (setns(fd, CLONE_NEWPID) == -1)
		errExit("setns");
	close(fd);
}

void print_child_handler(int sig) {
	int wstatus;
	pid_t pid;
	// https://man7.org/linux/man-pages/man2/waitpid.2.html
	// 获取子进程退出情况
	while ((pid=waitpid(-1, &wstatus, WNOHANG)) > 0) {
		printf("*** pid %d exit by %d signal\n", pid, WTERMSIG(wstatus));
	}
}

void register_signal_handler() {
	// 处理 SIGCHLD 信号，解决僵尸进程阻塞 Namespace 进程退出的情况。
	signal(SIGCHLD, print_child_handler);
}

int main(int argc, char *argv[])
{
	// seq: 0s
	printf("=== main: %d\n", getpid());
	// 注册 SIGCHLD 处理程序，会产生僵尸进程，而导致 PID Namespace 无法退出
	register_signal_handler();
	// 为子进程提供申请函数栈
	void *child_stack = mmap(NULL, STACK_SIZE,
							 PROT_READ | PROT_WRITE,
							 MAP_PRIVATE | MAP_ANONYMOUS | MAP_STACK,
							 -1, 0);
	if (child_stack == MAP_FAILED)
		errExit("mmap");
	// 创建新进程，并为该进程创建一个 PID Namespace（CLONE_NEWPID），并执行 new_namespace_func 函数
	// clone 库函数声明为：
	// int clone(int (*fn)(void *), void *stack, int flags, void *arg, ...
	// 		  /* pid_t *parent_tid, void *tls, pid_t *child_tid */);
	// 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
	pid_t pa = clone(new_namespace_func, child_stack + STACK_SIZE, SIGCHLD | CLONE_NEWNS | CLONE_NEWPID, NULL); // 进程 a
	if (pa == -1)
		errExit("clone-PA");
	printf("=== PA: %d\n", pa);

	my_sleep(1);
	// seq: 1s

	// 构造 进程 b
	char buf[256];
	// 通过 nsenter 进入进程 a 的 PID Namespace
	sprintf(buf, "exec nsenter -p -t %d bash -c 'echo === PB: \"$$ in new pid namespace\" && exec sleep infinity'", pa);
	proccess_b_args[2] = buf;
	pid_t pbp = fork_proccess(proccess_b_args);
	if (pbp == -1)
		errExit("clone-PB");
	my_sleep(1);

	// seq: 2s

	// 此时 kill 掉 nsenter 进程，sleep infinity 就能称为满足条件的进程 b
	kill(pbp, SIGKILL);

	// 主进程 setns PID Namespace 为 进程 a
	set_pid_namespace(pa);
	// fork 进程 c
	pid_t pc = fork_proccess(proccess_c_args);
	if (pc == -1)
		errExit("clone-PC");
	printf("=== PC: %d\n", pc);

	my_sleep(2);
	// seq: 4s

	// 恢复主进程 PID Namespace
	set_pid_namespace(1);
	printf("=== old pid namespace process ===\n");
	pid_t pe = fork_proccess(proccess_e_args);

	my_sleep(1);
	// seq: 5s

	return 0;
}
