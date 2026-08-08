/*
 * Minimal C harness for the perfmonger C-embedding contract (#0009 D9).
 *
 * Usage: embedding_harness <perfmonger-bin> <output-path>
 *
 * It creates a ready pipe and a stop pipe, forks, and in the child dup2()s
 * them onto fixed descriptors (ready write end -> 3, stop read end -> 4)
 * before execvp()ing:
 *
 *   <perfmonger-bin> record -l <output-path> -i 0.1 --ready-fd 3 --stop-fd 4
 *
 * The parent reads the one-line ready JSON event, prints it to stdout so the
 * test can inspect it, then closes the stop pipe's write end to stop the
 * recording, waitpid()s, and exits 0 only if everything lined up.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/wait.h>

#define READY_FD 3
#define STOP_FD 4

int main(int argc, char **argv) {
    if (argc != 3) {
        fprintf(stderr, "usage: %s <perfmonger-bin> <output-path>\n", argv[0]);
        return 2;
    }
    const char *bin = argv[1];
    const char *out = argv[2];

    int ready[2], stop[2];
    if (pipe(ready) != 0 || pipe(stop) != 0) {
        perror("pipe");
        return 2;
    }

    pid_t pid = fork();
    if (pid < 0) {
        perror("fork");
        return 2;
    }

    if (pid == 0) {
        /* Child: place the ready write end and stop read end on fixed fds.
         * The pipe descriptors may already occupy the target fds (3, 4), so
         * close the ends we do not need first and guard the remaining closes
         * against the dup2 targets to avoid clobbering what we just wired up. */
        close(ready[0]);  /* read end of ready pipe: unused in child */
        close(stop[1]);   /* write end of stop pipe: unused in child  */
        if ((ready[1] != READY_FD && dup2(ready[1], READY_FD) < 0) ||
            (stop[0] != STOP_FD && dup2(stop[0], STOP_FD) < 0)) {
            perror("dup2");
            _exit(127);
        }
        if (ready[1] != READY_FD && ready[1] != STOP_FD) close(ready[1]);
        if (stop[0] != READY_FD && stop[0] != STOP_FD) close(stop[0]);
        execlp(bin, bin, "record",
               "-l", out,
               "-i", "0.1",
               "--ready-fd", "3",
               "--stop-fd", "4",
               "--meta", "harness=c",
               (char *)NULL);
        perror("execlp");
        _exit(127);
    }

    /* Parent: close the child-owned ends so EOF/close semantics work. */
    close(ready[1]);
    close(stop[0]);

    /* Block until the recorder emits its ready line. */
    char buf[1024];
    ssize_t n = read(ready[0], buf, sizeof(buf) - 1);
    close(ready[0]);
    if (n <= 0) {
        fprintf(stderr, "harness: no ready event received\n");
        close(stop[1]);
        waitpid(pid, NULL, 0);
        return 3;
    }
    buf[n] = '\0';
    /* Emit the ready line for the test to parse. */
    fputs(buf, stdout);
    fflush(stdout);

    if (strstr(buf, "\"event\":\"ready\"") == NULL) {
        fprintf(stderr, "harness: ready line missing event=ready: %s\n", buf);
        close(stop[1]);
        waitpid(pid, NULL, 0);
        return 4;
    }

    /* Stop recording by closing the stop pipe's write end (child sees EOF). */
    close(stop[1]);

    int status = 0;
    if (waitpid(pid, &status, 0) < 0) {
        perror("waitpid");
        return 5;
    }
    if (!WIFEXITED(status)) {
        fprintf(stderr, "harness: child did not exit normally\n");
        return 6;
    }
    if (WEXITSTATUS(status) != 0) {
        fprintf(stderr, "harness: child exited %d\n", WEXITSTATUS(status));
        return WEXITSTATUS(status);
    }
    return 0;
}
