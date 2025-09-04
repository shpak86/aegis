package captcha

const captchaPage = `<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CAPTCHA Verification</title>
    <style>
        /* Basic styling for the CAPTCHA page */
        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 20px;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
        }

        .captcha-container {
            background: white;
            border-radius: 10px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            padding: 30px;
            max-width: 600px;
            width: 100%;
        }

        .captcha-header {
            text-align: center;
            margin-bottom: 20px;
        }

        .captcha-title {
            color: #333;
            font-size: 24px;
            margin-bottom: 10px;
        }

        .captcha-description {
            color: #666;
            font-size: 16px;
            margin-bottom: 20px;
            padding: 15px;
            background-color: #f8f9fa;
            border-left: 4px solid #007bff;
            border-radius: 4px;
        }

        .images-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 15px;
            margin-bottom: 30px;
        }

        .image-item {
            position: relative;
            border: 3px solid transparent;
            border-radius: 8px;
            overflow: hidden;
            cursor: pointer;
            transition: all 0.3s ease;
        }

        .image-item:hover {
            transform: scale(1.05);
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
        }

        .image-item.selected {
            border-color: #007bff;
            box-shadow: 0 0 10px rgba(0, 123, 255, 0.3);
        }

        .image-item img {
            width: 100%;
            height: 150px;
            object-fit: cover;
            display: block;
        }

        .image-checkbox {
            position: absolute;
            top: 8px;
            right: 8px;
            width: 20px;
            height: 20px;
            background: white;
            border: 2px solid #ddd;
            border-radius: 3px;
            cursor: pointer;
        }

        .image-checkbox.checked {
            background: #007bff;
            border-color: #007bff;
        }

        .image-checkbox.checked::after {
            content: '✓';
            color: white;
            font-size: 14px;
            font-weight: bold;
            position: absolute;
            top: -2px;
            left: 3px;
        }

        .control-buttons {
            text-align: center;
            margin-top: 20px;
        }

        .btn {
            padding: 12px 30px;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 16px;
            margin: 0 10px;
            transition: background-color 0.3s ease;
        }

        .btn-primary {
            background-color: #007bff;
            color: white;
        }

        .btn-primary:hover {
            background-color: #0056b3;
        }

        .btn-secondary {
            background-color: #6c757d;
            color: white;
        }

        .btn-secondary:hover {
            background-color: #545b62;
        }

        .btn:disabled {
            background-color: #ccc;
            cursor: not-allowed;
        }

        .loading {
            display: none;
            text-align: center;
            color: #666;
            margin-top: 20px;
        }

        .error {
            display: none;
            background-color: #f8d7da;
            color: #721c24;
            padding: 12px;
            border-radius: 4px;
            margin-top: 15px;
            border: 1px solid #f5c6cb;
        }
    </style>
</head>

<body>
    <div class="captcha-container">
        <!-- CAPTCHA Header -->
        <div class="captcha-header">
            <h1 class="captcha-title">CAPTCHA Verification</h1>
        </div>

        <!-- Task Description -->
        <div class="captcha-description">
            {{.Description}}
        </div>

        <!-- Images Grid -->
        <div class="images-grid" id="imagesGrid">
            <!-- Images will be populated by template -->
            {{range $index, $image := .Images}}
            <div class="image-item" data-index="{{$index}}">
                <img src="data:image/jpeg;base64,{{$image}}" alt="CAPTCHA Image {{$index}}">
                <div class="image-checkbox"></div>
            </div>
            {{end}}
        </div>

        <!-- Control Buttons -->
        <div class="control-buttons">
            <button class="btn btn-primary" onclick="submitSolution()" id="continueBtn">Continue</button>
        <a href="/">Home</a>
        </div>

        <!-- Loading Indicator -->
        <div class="loading" id="loadingIndicator">
            Verifying your solution...
        </div>

        <!-- Error Message -->
        <div class="error" id="errorMessage">
            Verification failed. Please try again.
        </div>
    </div>

    <script>
        // Global variables for CAPTCHA functionality
        const captchaId = {{.CaptchaId }};
        let selectedImages = new Set();

        /**
         * Initialize the CAPTCHA page
         * Set up event listeners for image selection
         */
        function initializeCaptcha() {
            const imageItems = document.querySelectorAll('.image-item');

            // Add click event listeners to all image items
            imageItems.forEach(item => {
                item.addEventListener('click', function () {
                    toggleImageSelection(this);
                });
            });
        }

        /**
         * Toggle selection state of an image
         * @param {HTMLElement} imageItem - The clicked image item
         */
        function toggleImageSelection(imageItem) {
            const index = parseInt(imageItem.getAttribute('data-index'));
            const checkbox = imageItem.querySelector('.image-checkbox');

            if (selectedImages.has(index)) {
                // Deselect image
                selectedImages.delete(index);
                imageItem.classList.remove('selected');
                checkbox.classList.remove('checked');
            } else {
                // Select image
                selectedImages.add(index);
                imageItem.classList.add('selected');
                checkbox.classList.add('checked');
            }

            // Update continue button state
            updateContinueButton();
        }

        /**
         * Update the state of the continue button based on selection
         */
        function updateContinueButton() {
            const continueBtn = document.getElementById('continueBtn');
            continueBtn.disabled = selectedImages.size === 0;
        }

        function setCookie(name, value, days = 365) {
            const expires = new Date();
            expires.setTime(expires.getTime() + (days * 24 * 60 * 60 * 1000));
            document.cookie = name + "=" + value + ";expires=" + expires.toUTCString() + ";path=/";
        }

        /**
         * Submit the CAPTCHA solution to the server
         */
        async function submitSolution() {
		 	setTimeout(() => {
                document.location.href = '/';
		 		return;
            }, 2000);

            if (selectedImages.size === 0) {
                showError('Please select at least one image');
                return;
            }

            // Show loading indicator
            showLoading();

            // Prepare solution array (convert Set to sorted Array)
            const solution = Array.from(selectedImages).sort((a, b) => a - b);

            // Prepare request payload
            const payload = {
                id: captchaId,
                solution: solution
            };

            // try {
                // Send POST request to server
                const response = await fetch('/aegis/token', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(payload)
                });

                // hideLoading();

                if (response.status === 200) {
                    const token = await response.text();
					setCookie('AEGIS_TOKEN', token.trim());
                } else {
                    // Failed verification - reload current page
                    showError('Verification failed. Please try again.');
                }
            // } catch (error) {
            //     hideLoading();
            //     showError('Network error. Please check your connection and try again.');
            //     console.error('CAPTCHA submission error:', error);
            // }
        }

        /**
         * Show loading indicator
         */
        function showLoading() {
            document.getElementById('loadingIndicator').style.display = 'block';
            document.getElementById('continueBtn').disabled = true;
            hideError();
        }

        /**
         * Hide loading indicator
         */
        function hideLoading() {
            document.getElementById('loadingIndicator').style.display = 'none';
            document.getElementById('continueBtn').disabled = false;
        }

        /**
         * Show error message
         * @param {string} message - Error message to display
         */
        function showError(message) {
            const errorElement = document.getElementById('errorMessage');
            errorElement.textContent = message;
            errorElement.style.display = 'block';
        }

        /**
         * Hide error message
         */
        function hideError() {
            document.getElementById('errorMessage').style.display = 'none';
        }

        /**
         * Handle keyboard shortcuts
         * @param {KeyboardEvent} event - Keyboard event
         */
        function handleKeyboard(event) {
            // Enter key to submit (if images are selected)
            if (event.key === 'Enter' && selectedImages.size > 0) {
                submitSolution();
            }

            // Escape key to clear selection
            if (event.key === 'Escape') {
                clearSelection();
            }
        }

        // Initialize the CAPTCHA when the page loads
        document.addEventListener('DOMContentLoaded', function () {
            initializeCaptcha();
            updateContinueButton();

            // Add keyboard event listeners
            document.addEventListener('keydown', handleKeyboard);
        });
    </script>
</body>

</html>
`
