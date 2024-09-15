package ui

var DBpage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>pal - DB</title>
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
      <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarsExample07XL" aria-controls="navbarsExample07XL" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>
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
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/files">
                <span class="material-symbols-outlined me-2">description</span>
                Files
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link active d-flex fw-bolder" href="/v1/pal/ui/db">
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
              <h5 class="card-title">Add Key-Value Pair</h5>
                <form method="post" action="/v1/pal/ui/db/put"> 
                  <div class="row fs-5">
                    <div class="col-md-5 mb-3">
                      <label for="keyInput" class="form-label"><strong>Key</strong></label>
                      <input type="text" class="form-control" id="keyInput" name="key" placeholder="Key" required>
                    </div>
                    <div class="col-md-5 mb-3">
                      <label for="valueInput" class="form-label"><strong>Value</strong></label>
                      <input type="text" class="form-control" id="valueInput" name="value" placeholder="Value" required>
                    </div>
                    <div class="col-md-2 d-flex align-items-end"> 
                      <button type="submit" class="btn btn-primary mb-3">Submit</button>
                    </div>
                  </div> 
                </form>
            </div>
          </div>
        </div>

        <div class="col-12 col-lg-12">
          <div class="card">
            <div class="card-body">
              <div class="table-responsive mt-3">
                <table class="table table-striped table-hover table-lg table-borderless mb-0 fs-5">
                  <thead>
                    <tr>
                      <th>Key</th>
                      <th>Value</th>
                      <th class="text-center">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {{range $key, $value := .}}
                    <tr>
                      <td><strong>{{$key}}</strong></td>
                      <td class="align-left"><pre>{{$value}}</pre></td>
                      <td class="text-center fs-5">
                	      <a href="/v1/pal/ui/db/delete?key={{$key}}" class="text-white"><button class="btn btn-sm btn-danger">
                          <span class="material-symbols-outlined align-bottom">
                            delete
                          </span>
                          <strong>Delete</strong>
                        </a></button>				
                      </td>
                    </tr>
                    {{end}}
                    </tr>
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
