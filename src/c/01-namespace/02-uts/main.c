#define _GNU_SOURCE    // Required for enabling clone(2)
#include <sys/types.h> // For wait(2)
#include <sys/wait.h>  // For wait(2)
#include <sys/mman.h>  // For mmap(2)
#include <sched.h>	   // For clone(2)
#include <signal.h>	   // For SIGCHLD constant
#include <stdio.h>     // For perror(3)
#include <unistd.h>
#include <stdlib.h>

#define STACK_SIZE (1024 * 1024)

char *const child_args[] = {
	"/bin/bash",
	NULL};

int child_main(void *args)
{
	execv(child_args[0], child_args);
	perror("exec");
	exit(EXIT_FAILURE);
}

int main()
{
	void *child_stack = mmap(NULL, STACK_SIZE,
							 PROT_READ | PROT_WRITE,
							 MAP_PRIVATE | MAP_ANONYMOUS | MAP_STACK,
							 -1, 0);
	pid_t p = clone(child_main, child_stack + STACK_SIZE, SIGCHLD | CLONE_NEWUTS, NULL);
	if (p == -1)
	{
		perror("clone");
		exit(1);
	}
	waitpid(p, NULL, 0);
	return 0;
}