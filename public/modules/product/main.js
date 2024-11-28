(() => {
	const combinations = []
	const borderNone = "border-transparent"
	const borderColor = "border-sky-500"
	$(function() {
		try {
			const $combinations = $("#data-combinations")
			if ($combinations.length <= 0) {
				throw new Error("couldnt find combinations within the document")
			}
			$combinations.find("combination").each(function() {
				const $combination = $(this)
				let comb = {}
				comb.sku = $combination.attr("sku") || ""
				comb.options = JSON.parse($combination.attr("options")).options || []
				combinations.push(comb)
			})
			if (combinations.length > 0) {
				const $combinations = $("#product-combinations")
				const $variants = $("#product-variants")
				const comb = combinations[combinations.length - 1]
				const sku = comb.sku
				showCombination($combinations, sku)
				showCurrentPicture(sku)
				normalizePictureSelector(sku)
				$variants.find('options option').each(function() {
					const $option = $(this)
					$option.removeClass(borderColor)
					$option.addClass(borderNone)
					$option.attr("data-selected", "false")
				});
				const options = comb.options
				for (const opt of options) {
					const option = opt.option
					$variants.
						find(`options
							option[option-id="${option.id}"][variant-id="${option.variantId}"]`).
						attr("data-selected", "true")
				}
				removeOptionsFocus($variants)
				normalizeSelectedOptions($variants)
			}
			parseOptions()
		} catch (err) {
			console.error(err)
		}

		$("#product-variants").on("click", "options option", function() {
			const $variants = $("#product-variants")
			$variants.find("options option").removeAttr("disabled")
			if (!combinations || combinations.length <= 0) {
				console.error("combinations not present or has no items")
				return
			}
			const $option = $(this)
			const selected = (($option.attr("data-selected") || "false") === "true")
			$option.closest("options").find("option").each(function() {
				$(this).attr("data-selected", false)
			});
			$option.attr("data-selected", (!(selected === true)).toString())
			parseOptions()
		});
	});

	function parseOptions() {
		const $variants = $("#product-variants")
		const $combinations = $("#product-combinations")
		const optionsSelected = []
		$variants.find('options option[data-selected="true"]').
			each(function() {
				const $option = $(this)
				const option = { label: "", id: 0, variantId: 0 }
				option.label = $option.attr("option-label")
				option.id = parseInt($option.attr("option-id") || 0)
				option.variantId = parseInt($option.attr("variant-id") || 0)
				optionsSelected.push(option)
			});
		(function valid() {
			if (!optionsSelected || optionsSelected.length <= 0) {
				$variants.find('options option').each(function() {
					const $option = $(this)
					$option.removeClass(borderColor)
					$option.addClass(borderNone)
				});
			}
			if (!combinations || combinations.length <= 0) {
				throw new Error("combinations not present or has no items")
			}
			let found = false
			let sku = ""
			for (const comb of combinations) {
				/** @type(any[]) */
				const options = comb.options
				if (
					!(optionsSelected.length === options.length) ||
					!optionsSelected.every((opt) => options.some((combOps) => {
						return combOps.option.variantId === opt.variantId &&
							combOps.option.id === opt.id
					}))
				) {
					continue
				}
				sku = comb.sku
				found = true
			}
			if (!found) {
				$variants.find("options option[data-selected='true']").attr("disabled", "true")
				$variants.find('options option').each(function() {
					const $option = $(this)
					$option.removeClass(borderColor)
					$option.addClass(borderNone)
				});
				return
			}
			removeOptionsFocus($variants)
			normalizeSelectedOptions($variants)
			showCurrentPicture(sku)
			normalizePictureSelector(sku)
			showCombination($combinations, sku)
		})();
	}
	/** @param {Jquery<HTMLElement>} $variants */
	function removeOptionsFocus($variants) {
		$variants.find('options option').each(function() {
			const $option = $(this)
			$option.removeClass(borderColor)
			$option.addClass(borderNone)
		});
	}
	/** @param {Jquery<HTMLElement>} $variants */
	function normalizeSelectedOptions($variants) {
		$variants.find('options option[data-selected="true"]').
			each(function() {
				const $option = $(this)
				$option.removeClass(borderNone)
				$option.addClass(borderColor)
			});
	}
	/** @param {string} sku */
	function showCurrentPicture(sku) {
		if ($(`#gallery picture-container[sku="${sku}"]`).length > 0) {
			$(`#gallery picture-container`).addClass("hidden")
			$(`#gallery picture-container[sku="${sku}"]`).removeClass("hidden")
		}
	}
	/** @param {string} sku */
	function normalizePictureSelector(sku) {
		if ($(`#picture-selector picture-container[sku="${sku}"]`).length > 0) {
			$(`#picture-selector picture-container`).
				addClass(borderNone).removeClass(borderColor)
			$(`#picture-selector picture-container[sku="${sku}"]`).
				removeClass(borderNone).addClass(borderColor)
		}

	}
	/** @param {JQuery<HTMLElement} $combinations @param {string} sku */
	function showCombination($combinations, sku) {
		if ($combinations.find(`combination[sku="${sku}"]`).length > 0) {
			$combinations.find(`combination`).addClass("hidden")
			$combinations.find(`combination[sku="${sku}"]`).removeClass("hidden")
		}
	}
})();
