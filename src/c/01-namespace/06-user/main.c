// gcc src/c/01-namespace/06-user/main.c && ./a.out

#define _GNU_SOURCE	   // Required for enabling clone(2)
#include <sys/wait.h>  // For waitpid(2)
#include <sys/mount.h> // For mount(2)
#include <sys/mman.h>  // For mmap(2)
#include <sys/syscall.h> // For SYS_pidfd_open

#include <time.h>	   // For nanosleep(2)
#include <sched.h>	   // For clone(2)
#include <signal.h>	   // For SIGCHLD
#include <stdio.h>	   // For perror(3), printf(3), perror(3)
#include <unistd.h>    // For execv(3), sleep(3), read(2)
#include <stdlib.h>	   // For exit(3), system(3), free(3), realloc(3)
#include <errno.h>	   // For open(2), O_RDONLY
#include <string.h>	   // For strtok(3)

#define errExit(msg)    do { perror(msg); exit(EXIT_FAILURE); \
                               } while (0)

#define STACK_SIZE (1024 * 1024)

// https://stackoverflow.com/a/44894946
/* Size of each input chunk to be
   read and allocate for. */

#define  READALL_CHUNK  4096
#define  READALL_OK          0  /* Success */
#define  READALL_INVALID    -1  /* Invalid parameters */
#define  READALL_ERROR      -2  /* Stream error */
#define  READALL_TOOMUCH    -3  /* Too much input */
#define  READALL_NOMEM      -4  /* Out of memory */

/* This function returns one of the READALL_ constants above.
   If the return value is zero == READALL_OK, then:
     (*dataptr) points to a dynamically allocated buffer, with
     (*sizeptr) chars read from the file.
     The buffer is allocated for one extra char, which is NUL,
     and automatically appended after the data.
   Initial values of (*dataptr) and (*sizeptr) are ignored.
*/
int readall(FILE *in, char **dataptr, size_t *sizeptr)
{
    char  *data = NULL, *temp;
    size_t size = 0;
    size_t used = 0;
    size_t n;

    /* None of the parameters can be NULL. */
    if (in == NULL || dataptr == NULL || sizeptr == NULL)
        return READALL_INVALID;

    /* A read error already occurred? */
    if (ferror(in))
        return READALL_ERROR;

    while (1) {

        if (used + READALL_CHUNK + 1 > size) {
            size = used + READALL_CHUNK + 1;

            /* Overflow check. Some ANSI C compilers
               may optimize this away, though. */
            if (size <= used) {
                free(data);
                return READALL_TOOMUCH;
            }

            temp = realloc(data, size);
            if (temp == NULL) {
                free(data);
                return READALL_NOMEM;
            }
            data = temp;
        }

        n = fread(data + used, 1, READALL_CHUNK, in);
        if (n == 0)
            break;

        used += n;
    }

    if (ferror(in)) {
        free(data);
        return READALL_ERROR;
    }

    temp = realloc(data, used + 1);
    if (temp == NULL) {
        free(data);
        return READALL_NOMEM;
    }
    data = temp;
    data[used] = '\0';

    *dataptr = data;
    *sizeptr = used;

    return READALL_OK;
}

void print_caps() {
	FILE *f = fopen("/proc/self/status", "r");
	if (f == NULL)
		errExit("fopen");
	char *buf;
	size_t len;
	if (readall(f, &buf, &len) != READALL_OK)
		errExit("readall");
	fclose(f);

	char *delimiter = "\r\n";
	char *line = strtok(buf, delimiter);
	while (line != NULL) {
		char *pre = "Cap";
		if (strncmp(pre, line, strlen(pre)) == 0)
			printf("%s\n", line);
		line = strtok(NULL, delimiter);
	}
}

int new_namespace_func(void *args) {
	print_caps();
	return 0;
}

int main(int argc, char *argv[])
{
	new_namespace_func(NULL);
	if (unshare(CLONE_NEWUSER) < 0)
		errExit("unshare");
	new_namespace_func(NULL);
	return 0;
	// 为子进程提供申请函数栈
	void *child_stack = mmap(NULL, STACK_SIZE,
							 PROT_READ | PROT_WRITE,
							 MAP_PRIVATE | MAP_ANONYMOUS | MAP_STACK,
							 -1, 0);
	if (child_stack == MAP_FAILED)
		errExit("mmap");
	// 创建新进程，并为该进程创建一个 PID Namespace（CLONE_NEWUSER），并执行 new_namespace_func 函数
	// clone 库函数声明为：
	// int clone(int (*fn)(void *), void *stack, int flags, void *arg, ...
	// 		  /* pid_t *parent_tid, void *tls, pid_t *child_tid */);
	// 更多参见：https://man7.org/linux/man-pages/man2/clone.2.html
	pid_t p = clone(new_namespace_func, child_stack + STACK_SIZE, SIGCHLD | CLONE_NEWUSER, NULL);
	if (p < 0)
		errExit("clone");
	if (waitpid(p, NULL, 0) < 0)
		errExit("pid");
	return 0;
}
