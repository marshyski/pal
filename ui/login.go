package ui

var LoginPage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>pal - Login</title>
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
    <link rel="icon" type="image/svg+xml" href="/favicon.ico">
  </head>
  <body class="bg-light">
    <div
      class="container d-flex justify-content-center align-items-center vh-100"
    >
      <div class="card p-5">
        <div class="card-body">
          <div class="card p-5 shadow-lg">
            <div class="card-body">
              <div class="mb-3">
                <h1
                  class="card-title text-center mb-4"
                  style="font-family: Sixtyfour, sans-serif"
                >
                  pal
                </h1>
                <form action="/v1/pal/ui/login" method="post">
                  <div class="mb-3">
                    <label for="username" class="form-label"
                      ><strong>Username</strong></label
                    >
                    <input
                      type="text"
                      class="form-control"
                      placeholder="Username"
                      id="username"
                      name="username"
                    />
                  </div>
                  <div class="mb-3">
                    <label for="password" class="form-label"
                      ><strong>Password</strong></label
                    >
                    <input
                      type="password"
                      class="form-control"
                      placeholder="Password"
                      id="password"
                      name="password"
                    />
                  </div>
                  <br />
                  <button type="submit" class="btn btn-md btn-info btn-primary w-100">
                    <strong>Login</strong>
                  </button>
                </form>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </body>
</html>`
