(() => {
  const codesAPI = (() => {
    let token = {};
    let timeout;
    return {
      setToken: function(tk) {
        clearTimeout(timeout)
        token = tk;
        timeout = setTimeout(() => { token = {} }, (60000 * 15))
      },
      getToken: function() {
        const tk = token
        token = {}
        return tk;
      }
    };
  })();
  const cartAPI = (() => {
    let cart = {};
    let timeout;
    return {
      setCart: function(dataCart) {
        clearTimeout(timeout)
        cart = dataCart
        timeout = setTimeout(() => { cart = {} }, (60000 * 15))
      },
      getCart: function() {
        const c = JSON.parse(JSON.stringify(cart))
        cart = {}
        return c
      }
    };
  })();

  paypal
    .Buttons({
      onInit(_, actions) {
        $("#loading-indicator").removeClass("something")
        actions.enable()
      },
      async createOrder() {
        try {
          const products = []
          const fromCartAttrb = $("#cart-items").attr("from-cart")
          const fromCart = fromCartAttrb !== undefined && fromCartAttrb !== false
          $("#cart-items product").each(function(_) {
            const product = $(this)
            products.push({ sku: product.attr("sku"), quantity: parseInt(product.attr("quantity")) })
          })
          const response = await fetch("/create-paypal-order", {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              products: [
                ...products
              ],
              currency: "USD",
              fromCart: fromCart
            }),
          });
          let order = await response.text();
          if (order.length <= 0 || (order[0] != "[" && order[0] != "{")) {
            throw new Error("invalid order format, not a JSON")
          }
          const accessToken = parseToken(response.headers.get("Authorization"))
          if (!accessToken || accessToken.length <= 0) {
            throw new Error("response has no token")
          }
          const referenceId = response.headers.get("Reference-Id")
          if (!referenceId || referenceId.length <= 0) {
            throw new Error("response has no reference id")
          }
          order = JSON.parse(order)
          if (order.id) {
            cartAPI.setCart({
              products: [
                ...products
              ],
              currency: "USD",
              fromCart: fromCart
            });
            codesAPI.setToken({ accessToken: accessToken, referenceId: referenceId })
            return order.id
          } else {
            const errorDetail = order?.details?.[0];
            const errorMessage = errorDetail
              ? `${errorDetail.issue} ${errorDetail.description} (${order.debug_id})`
              : JSON.stringify(order);

            throw new Error(errorMessage);
          }
        } catch (err) {
          console.error(err);
        }
      },
      async onApprove(data, actions) {
        let response
        try {
          const tokens = codesAPI.getToken()
          const referenceId = tokens.referenceId
          response = await fetch(`/capture-paypal-order/${data.orderID}`, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "Authorization": `Bearer ${tokens.accessToken}`,
              "Reference-Id": referenceId,
              "HX-Request": "true"
            },
            body: JSON.stringify(cartAPI.getCart())
          });

          const orderData = await response.json();
          const errorDetail = orderData?.details?.[0];
          if (errorDetail?.issue === "INSTRUMENT_DECLINED") {
            return actions.restart();
          } else if (errorDetail) {
            redirect(response)
            throw new Error(`${errorDetail.description} (${orderData.debug_id})`);
          } else if (!orderData.purchase_units) {
            redirect(response)
            throw new Error(JSON.stringify(orderData));
          } else {
            redirect(response)
            const transaction =
              orderData?.purchase_units?.[0]?.payments?.captures?.[0] ||
              orderData?.purchase_units?.[0]?.payments?.authorizations?.[0];
            console.log(
              `Transaction ${transaction.status}: ${transaction.id}<br><br>See console for all available details`,
            );
            console.log(
              "Capture result",
              orderData,
              JSON.stringify(orderData, null, 2),
            );
          }
        } catch (error) {
          redirect(response)
          console.error(error);
        }
      },
    })
    .render("#paypal-button-container");

  /** 
    * @param {string} token 
    * @returns {string}
  */
  function parseToken(token) {
    let Atoken = token.split("Bearer ")
    if (Atoken.length <= 1) {
      return ""
    }
    return Atoken[1]
  }
  function redirect(res) {
    const url = res.headers.get("HX-Redirect")
    if (url.length > 0) {
      document.location.replace(url)
    }
  }
})();
