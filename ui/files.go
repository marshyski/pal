package ui

var FilesPage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>pal - Files</title>
    <link
      rel="stylesheet"
      href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css"
    />
    <link
      rel="stylesheet"
      href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@48,700,1,200"
    />
    <link
      rel="stylesheet"
      href="https://fonts.googleapis.com/css?family=Sixtyfour"
    />
    <link rel="stylesheet" href="/v1/pal/ui/main.css" />
  </head>
  <body>
    <nav
      class="navbar navbar-expand-lg fixed-top navbar-dark bg-dark"
      aria-label="Main navigation"
    >
      <div class="container-xl">
        <a
          class="navbar-brand fs-3"
          style="font-family: Sixtyfour, sans-serif"
          href="/v1/pal/ui"
          >pal</a
        >
        <div class="collapse navbar-collapse" id="navbarsExample07XL">
          <ul class="navbar-nav ms-auto mb-2 mb-lg-0">
            <li class="nav-item">
              <a
                class="nav-link d-flex fw-bolder"
                aria-current="page"
                href="/v1/pal/ui"
              >
                <span class="material-symbols-outlined me-2"
                  >rule_settings</span
                >
                Actions
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/notifications">
                <span class="material-symbols-outlined me-2">inbox</span>
                Notifications
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/schedules">
                <span class="material-symbols-outlined me-2">schedule</span>
                Schedules
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link active d-flex fw-bolder" href="/v1/pal/ui/files">
                <span class="material-symbols-outlined me-2">description</span>
                Files
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/db">
                <span class="material-symbols-outlined me-2">database</span>
                DB
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/logout">
                <span class="material-symbols-outlined me-2">logout</span>
                Logout
              </a>
            </li>
          </ul>
        </div>
      </div>
    </nav>
    <main class="container">
      <div class="row">
        <div class="col-12 col-lg-12">
          <div class="card">
            <div class="card-body">
              <h5 class="card-title">Upload Files</h5>
              <form enctype="multipart/form-data" method="post" action="/v1/pal/ui/files/upload">
              <div class="mb-3 d-flex align-items-center">
                <input class="form-control flex-grow-1" type="file" id="fileInput" name="files" multiple />
                <button type="submit" class="btn btn-sm btn-primary ms-4">
                  <span class="material-symbols-outlined align-bottom">cloud_upload</span>
                  <strong>Upload</strong>
                </button>
              </div>
              </form>
            </div>
          </div>
        </div>

        <div class="col-12 col-lg-12">
          <div class="card">
            <div class="card-body">
              <h5 class="mb-3">Directory Listing</h5>
              <div class="table-responsive mt-3">
                <table class="table table-striped table-hover table-lg table-borderless mb-0 fs-5">
                  <thead>
                    <tr>
                      <th>Filename</th>
                      <th>Size</th>
                      <th>Last Modified</th>
                      <th class="text-center">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
				            {{range .Files}}
                    <tr>
                      <td><a href="/v1/pal/ui/files/download/{{.Name}}">{{.Name}}</a></td>
                      <td>{{fileSize .}}</td>
                      <td>{{fileModTime .}}</td>
                      <td class="text-center">
					              <a href="/v1/pal/ui/files/delete/{{.Name}}">
                          <button class="btn btn-sm btn-danger">
                            <span class="material-symbols-outlined align-bottom">
                              delete
                            </span>
                          <strong>Delete</strong>
                        </button>
					              </a>
                      </td>
                    </tr>
					          {{end}}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/js/bootstrap.bundle.min.js"></script>
  </body>
</html>`
