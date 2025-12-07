package models

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
}

func NewEmailService() (*EmailService, error) {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpHost == "" || smtpUser == "" || smtpPass == "" {
		return nil, fmt.Errorf("SMTP configuration missing")
	}

	port, err := strconv.Atoi(smtpPort)
	if err != nil {
		port = 587
	}

	dialer := gomail.NewDialer(smtpHost, port, smtpUser, smtpPass)

	return &EmailService{dialer: dialer}, nil
}

func (s *EmailService) SendOTPEmail(toEmail, otp string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_FROM"))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Password Reset OTP - Harlan Holden Coffee")

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #f97316; }
        .otp-box { background-color: #fff7ed; border: 2px dashed #f97316; padding: 20px; text-align: center; margin: 30px 0; border-radius: 8px; }
        .otp-code { font-size: 36px; font-weight: bold; color: #f97316; letter-spacing: 8px; }
        .footer { text-align: center; margin-top: 30px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">Harlan Holden Coffee</div>
        </div>
        <h2 style="color: #333;">Password Reset Request</h2>
        <p>Hello,</p>
        <p>You have requested to reset your password. Please use the following One-Time Password (OTP) to proceed:</p>
        
        <div class="otp-box">
            <div style="color: #666; font-size: 14px; margin-bottom: 10px;">Your OTP Code</div>
            <div class="otp-code">%s</div>
        </div>
        
        <p><strong>This code will expire in 5 minutes.</strong></p>
        <p>If you did not request a password reset, please ignore this email or contact support if you have concerns.</p>
        
        <div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
            <p style="color: #666; font-size: 14px;">Best regards,<br>Harlan Holden Coffee Team</p>
        </div>
        
        <div class="footer">
            <p>This is an automated email. Please do not reply.</p>
            <p>&copy; 2024 Harlan Holden Coffee. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
	`, otp)

	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *EmailService) SendOrderConfirmationEmail(toEmail, orderNumber string, total int) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_FROM"))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", fmt.Sprintf("Order Confirmation #%s - Harlan Holden Coffee", orderNumber))

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #f97316; }
        .order-box { background-color: #fff7ed; padding: 20px; margin: 20px 0; border-radius: 8px; }
        .footer { text-align: center; margin-top: 30px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">Harlan Holden Coffee</div>
        </div>
        <h2 style="color: #333;">Order Confirmation</h2>
        <p>Thank you for your order!</p>
        
        <div class="order-box">
            <p><strong>Order Number:</strong> %s</p>
            <p><strong>Total Amount:</strong> IDR %s</p>
        </div>
        
        <p>Your order has been received and is being processed. We'll notify you when your order is ready.</p>
        
        <div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
            <p style="color: #666; font-size: 14px;">Thank you for choosing us!<br>Harlan Holden Coffee Team</p>
        </div>
        
        <div class="footer">
            <p>&copy; 2024 Harlan Holden Coffee. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
	`, orderNumber, formatRupiah(total))

	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func formatRupiah(amount int) string {
	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		return str
	}

	result := ""
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(digit)
	}
	return result
}
