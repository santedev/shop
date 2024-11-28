$(function() {
	$("input[type='number']").each(function() {
		const initialValue = $(this).attr("value");
		$(this).val(initialValue);
	});
	function delay(fn, time) {
		let timeout
		return function(...args) {
			clearTimeout(timeout)
			timeout = setTimeout(() => fn.apply(this, args), time);
		}
	}
	$("#cart-products").on("input, keyup", "input[type='number']", delay(function() {
		const input = $(this)
		let count = parseInt(input.val()) || 1
		const min = parseInt(input.attr("min")) || 1
		const max = parseInt(input.attr("max")) || 99
		if (count < min) {
			count = min
		} else if (count > max) {
			count = max
		}
		input.val(count)
	}, 500));
	$("#cart-products").on("click", "minus, plus", function() {
		const mpHtml = $(this)
		let quantity
		if (mpHtml.is("minus")) {
			quantity = -1
		} else if (mpHtml.is("plus")) {
			quantity = 1
		} else {
			throw new Error("not the correct element")
		}
		const sku = mpHtml.attr("sku")
		const input = $(`input[sku="${sku}"]`)
		let count = (parseInt(input.val()) || 0) + quantity
		const min = parseInt(input.attr("min")) || 1
		const max = parseInt(input.attr("max")) || 99
		if (count < min) {
			count = min
		} else if (count > max) {
			count = max
		}
		input.val(count)
		htmx.trigger(`input[sku="${sku}"]`, 'change');
	});
})

