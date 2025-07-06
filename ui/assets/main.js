function sendData() {
  const data = document.getElementById("inputInput").value;
  const outputPre = document.getElementById("outputPre");
  const runIcon = document.getElementById("runIcon");
  const runURL = window.location.pathname + "/run";

  runIcon.classList.add("spin-animation");

  fetch(runURL, {
    method: "POST",
    headers: { "Content-Type": "text/plain" },
    body: data,
  })
    .then((response) => {
      const outputStatus = document.getElementById("outputStatus");
      if (!response.ok) {
        outputStatus.textContent = "error";
        outputStatus.style.color = "red";
        const status = response.status;
        const statusText = response.statusText;
        response
          .clone()
          .json()
          .then((errorData) => {
            const errorNotification = `Network response was not ok (Status ${status}: ${statusText}). Error details: ${JSON.stringify(
              errorData
            )}`;
            throw new Error(errorNotification);
          })
          .catch(() => {
            const errorNotification = `Network response was not ok (Status ${status}: ${statusText})`;
            throw new Error(errorNotification);
          });
      } else {
        outputStatus.textContent = "check_circle";
        outputStatus.style.color = "green";
      }

      const headersString = [...response.headers.entries()]
        .map(([key, value]) => `${key}: ${value}`)
        .join("\n");
      return response
        .text()
        .then((textData) => `${textData}\n\n${headersString}`);
    })
    .then((data) => {
      outputPre.textContent = data;
    })
    .catch((error) => {
      console.error(error);
      outputPre.textContent = error;
      const outputStatus = document.getElementById("outputStatus");
      outputStatus.textContent = "error";
      outputStatus.style.color = "red";
    })
    .finally(() => {
      runIcon.classList.remove("spin-animation");
    });
}

function actionsSendData(url, textareaInput, runIconId) {
  const runIcon = document.getElementById(runIconId);
  runIcon.classList.add("spin-animation");
  try {
    fetch(url, {
      method: "POST",
      headers: { "Content-Type": "text/plain" },
      body: textareaInput,
    })
      .then((response) => {
        if (!response.ok) {
          const status = response.status;
          const statusText = response.statusText;
          response
            .clone()
            .json()
            .then((errorData) => {
              const errorNotification = `Network response was not ok (Status ${status}: ${statusText}). Error details: ${JSON.stringify(
                errorData
              )}`;
              throw new Error(errorNotification);
            })
            .catch(() => {
              const errorNotification = `Network response was not ok (Status ${status}: ${statusText})`;
              throw new Error(errorNotification);
            });
        }
      })
      .then((data) => {
        outputPre.textContent = JSON.stringify(data, null, 2);
      })
      .catch((error) => {
        console.error(error);
        outputPre.textContent = error;
      })
      .finally(() => {
        runIcon.classList.remove("spin-animation");
        location.reload(true);
      });
  } catch (error) {
    console.error("Fetch request error:", error);
  }
}

function copyToClipboard() {
  const textToCopy = document.getElementById("outputPre").textContent;
  navigator.clipboard
    .writeText(textToCopy)
    .then(() => {
      const copiedTextElement = document.getElementById("copied-text");
      if (copiedTextElement) {
        copiedTextElement.textContent = "Copied!";
      }
    })
    .catch((err) => {
      console.error("Failed to copy text: ", err);
    });
}

document.addEventListener("DOMContentLoaded", () => {
  const runButton = document.getElementById("runNowBtn");
  if (runButton) {
    runButton.addEventListener("click", sendData);
  }

  const copyButton = document.getElementById("copyBtn");
  if (copyButton) {
    copyButton.addEventListener("click", copyToClipboard);
  }

  const actionButtons = document.querySelectorAll(".action-btn");
  actionButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const group = button.dataset.group;
      const action = button.dataset.action;

      const inputId = `input${group}${action}`;
      const runIconId = `runIcon${group}${action}`;
      const url = `/v1/pal/ui/action/${group}/${action}/run`;

      const inputValue = document.getElementById(inputId).value;

      actionsSendData(url, inputValue, runIconId);
    });
  });
});
