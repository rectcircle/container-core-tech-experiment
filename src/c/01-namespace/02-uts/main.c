// gcc src/c/01-namespace/02-uts/main.c && sudo ./a.out

#define _GNU_SOURCE	   // Required for enabling clone(2)
#include <sys/wait.h>  // For waitpid(2)
#include <sys/mount.h> // For mount(2)
#include <sys/mman.h>  // For mmap(2)
#include <sched.h>	   // For clone(2)
#include <signal.h>	   // For SIGCHLD constant
#include <stdio.h>	   // For perror(3), printf(3), perror(3)
#include <unistd.h>	   // For execv(3), sleep(3), sethostname(2), setdomainname(2)
#include <stdlib.h>    // For exit(3), system(3)
#include <string.h>    // For strlen(3)

#define errExit(msg)    do { perror(msg); exit(EXIT_FAILURE); \
                               } while (0)

#define STACK_SIZE (1024 * 1024)

char *const child_args[] = {
	"/bin/bash",
	"-xc",
	"hostname && hostname --nis || true",
	NULL};

char *const new_hostname = "new-hostname";
char *const new_domainname = "new-domainname";

int new_namespace_func(void *args)
{
	if (sethostname(new_hostname, strlen(new_hostname)) == -1 )
		errExit("sethostname");
	if (setdomainname(new_domainname, strlen(new_domainname)) == -1)
		errExit("setdomainname");
	printf("=== new uts namespace process ===\n");
	execv(child_args[0], child_args);
	perror("exec");
	exit(EXIT_FAILURE);
}

pid_t old_namespace_exec()
{
	pid_t p = fork();
	if (p == 0)
	{
		printf("=== old namespace process ===\n");
		execv(child_args[0], child_args);
		perror("exec");
		exit(EXIT_FAILURE);
	}
	return p;
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
	// 创建新进程，并为该进程创建一个 UTS Namespace（CLONE_NEWUTS），并执行 new_namespace_func 函数
	// clone 库函数声明为：
	// int clone(int (*fn)(void *), void *stack, int flags, void *arg, ...
	// 		  /* pid_t *parent_tid, void *tls, pid_t *child_tid */);
	// 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
	pid_t p1 = clone(new_namespace_func, child_stack + STACK_SIZE, SIGCHLD | CLONE_NEWUTS, NULL);
	if (p1 == -1)
		errExit("clone");
	sleep(5);
	// 创建新的进程（不创建 Namespace），并执行测试命令
	pid_t p2 = old_namespace_exec();
	if (p2 == -1)
		errExit("fork");
	waitpid(p1, NULL, 0);
	waitpid(p2, NULL, 0);
	return 0;
}
