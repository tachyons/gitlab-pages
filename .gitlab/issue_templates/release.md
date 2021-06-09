- [ ] Set the milestone on this issue
- Decide on the version number by reference to
    the [Versioning](https://gitlab.com/gitlab-org/gitlab-pages/blob/master/PROCESS.md#versioning)
    * Typically if you want to release code from current `master` branch you will update `MINOR` version, e.g. `1.12.0` -> `1.13.0`. In that case you **don't** need to create stable branch
    * If you want to backport some bug fix or security fix you will need to update stable branch `X-Y-stable`
- [ ] Create an MR for [gitlab-pages project](https://gitlab.com/gitlab-org/gitlab-pages).
    You can use [this MR](https://gitlab.com/gitlab-org/gitlab-pages/merge_requests/217) as an example.
    - [ ] Update `VERSION`, and push your branch
    - [ ] Update `CHANGELOG` by running `GITLAB_PRIVATE_TOKEN= make changelog`, note that you need to create a personal access token 
    - [ ] Assign to reviewer
- [ ] Once `gitlab-pages` is merged create a signed+annotated tag pointing to the **merge commit** on the **stable branch**
    In case of `master` branch:
    ```shell
    git fetch origin master
    git fetch dev master
    git tag -a -s -m "Release v1.0.0" v1.0.0 origin/master
    ```
    In case of `stable` branch:
    ```shell
    git fetch origin 1-0-stable
    git fetch dev 1-0-stable
    git tag -a -s -m "Release v1.0.0" v1.0.0 origin/1-0-stable
    ```
- [ ] Verify that you created tag properly:
    ```shell
    git show v1.0.0
    ```
    it should include something like:
    * ```(tag: v1.0.0, origin/master, dev/master, master)``` for `master`
    * ```(tag: v1.0.1, origin/1-0-stable, dev/1-0-stable, 1-0-stable)``` for `stable` branch
- [ ] Push this tag to origin(**Skip this for security release!**)
    ```shell
    git push origin v1.0.0
    ```
- [ ] Wait for tag to be mirrored to `dev` or push it:
    ```shell
    git push dev v1.0.0
    ```
- [ ] Create an MR for [gitlab project](https://gitlab.com/gitlab-org/gitlab).
    You can use [this MR](https://gitlab.com/gitlab-org/gitlab/merge_requests/23023) as an example.
    - [ ] Update `GITLAB_PAGES_VERSION`
    - [ ] Added `Changelog: added` footer to your commit
    - [ ] Assign to a reviewer

/label ~backend ~"Category:Pages" ~"devops::release" ~"group::release" ~"feature::maintenance"
