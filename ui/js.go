package ui

var MainJS = `function sendData() {
            const data = document.getElementById('inputInput').value;
            const outputPre = document.getElementById('outputPre');

			const runURL = window.location.pathname + "/run";

            fetch(runURL, {
                method: 'POST',
                headers: {
                    'Content-Type': 'text/plain'
                },
                body: data
            })
            .then(response => {
			    const outputStatus = document.getElementById('outputStatus');

                if (!response.ok) {
				    outputStatus.textContent = "error";
            		outputStatus.style.color = "red";
    				const status = response.status;
    				const statusText = response.statusText;
    				response.clone().json()
        				.then(errorData => {
            				const errorNotification = "Network response was not ok (Status " + status + ": " + statusText + "). Error details: " + JSON.stringify(errorData);
            				throw new Error(errorNotification);
        				})
        				.catch(() => {
							const errorNotification = "Network response was not ok (Status " + status + ": " + statusText + ")";
							throw new Error(errorNotification);
        				});
                } else {
            		outputStatus.textContent = "check_circle";
            		outputStatus.style.color = "green";				 
				}
                // Get all headers as an array
                const headersArray = [...response.headers.entries()];

                // Format headers into a string
				const headersString = headersArray.map(([key, value]) => key + ": " + value).join('\n');

                return response.text().then(textData => {
                    return textData + "\n\n" + headersString;
                });
            })
            .then(data => {
                outputPre.textContent = data; // Update the <pre> content
            })
            .catch(error => {
                console.error(error);
                outputPre.textContent = error; // Display error in <pre>
				outputStatus.textContent = "error";
        		outputStatus.style.color = "red";
            });
        }`
