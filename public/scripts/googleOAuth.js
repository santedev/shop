function googleOAuth(response) {
  fetch("/auth/google/idtoken", {
    method: "POST",
    body: JSON.stringify(response),
    headers: {
      "Content-Type": "application/json",
    },
  })
    .then((res) => {
      if (!res.ok) {
        return res.text().then((errorData) => {
          if (errorData[0] !== "[" && errorData[0] !== "{") {
            console.error("Error from server:", errorData);
            return;
          }
          const resJson = JSON.parse(errorData);
          if (resJson.redirect_url) {
            window.location.href = resJson.redirect_url;
          }
          console.error(resJson.context);
        });
      }
      return res.text().then((responseBody) => {
        const resJson = JSON.parse(responseBody);
        if (resJson.redirect_url) {
          window.location.href = resJson.redirect_url;
        }
      });
    })
    .catch((err) => {
      console.error(err);
    });
}
