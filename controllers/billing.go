package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/decentvcs/server/config"
	"github.com/decentvcs/server/lib/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/subscription"
	"go.mongodb.org/mongo-driver/bson"
)

// Get or create a new billing subscription.
func GetOrCreateBillingSubscription(c *fiber.Ctx) error {
	userData := auth.GetUserDataFromContext(c)
	stytchUser := auth.GetStytchUserFromContext(c)
	// team := team_lib.GetTeamFromContext(c)

	var stripeCustomer *stripe.Customer
	if userData.StripeCustomerID == "" {
		// Create a new Stripe customer for the user
		var customerName string
		if stytchUser.Name.FirstName == "" {
			customerName = userData.Username
		} else {
			customerName = fmt.Sprintf("%s %s", stytchUser.Name.FirstName, stytchUser.Name.LastName)
		}

		customerParams := &stripe.CustomerParams{
			Email: stripe.String(stytchUser.Emails[0].Email),
			Name:  stripe.String(customerName),
		}

		newStripeCustomer, err := customer.New(customerParams)
		if err != nil {
			fmt.Printf("Error creating Stripe customer for user %s: %+v\n", userData.ID, err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		stripeCustomer = newStripeCustomer

		// Update user data with Stripe customer ID
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		userData.StripeCustomerID = stripeCustomer.ID
		if _, err := config.MI.DB.Collection("user_data").UpdateByID(
			ctx,
			userData.ID,
			bson.M{
				"$set": bson.M{
					"stripe_customer_id": userData.StripeCustomerID,
				},
			},
		); err != nil {
			fmt.Printf("Error updating user data with Stripe customer ID: %+v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
	} else {
		// Get existing Stripe customer
		existingStripeCustomer, err := customer.Get(userData.StripeCustomerID, nil)
		if err != nil {
			fmt.Printf(
				"Error getting Stripe customer (%s) for user %s: %+v\n",
				userData.StripeCustomerID,
				userData.ID,
				err,
			)
			return c.Status(500).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		stripeCustomer = existingStripeCustomer
	}

	if userData.StripeSubscriptionID == "" {
		// Create Stripe subscription (incomplete)
		subParams := &stripe.SubscriptionParams{
			Customer: stripe.String(stripeCustomer.ID),
			Items: []*stripe.SubscriptionItemsParams{
				{
					Price: stripe.String(config.I.Stripe.CloudPlanPriceID),
					// Metadata: map[string]string{
					// 	"team_id": team.ID.Hex(),
					// },
				},
			},
			PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
				SaveDefaultPaymentMethod: stripe.String("on_subscription"),
			},
			PaymentBehavior: stripe.String("default_incomplete"),
		}
		subParams.AddExpand("latest_invoice.payment_intent")

		stripeSub, err := subscription.New(subParams)
		if err != nil {
			fmt.Printf("Error creating Stripe subscription for user %s: %+v\n", userData.ID, err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}

		// Return Stripe subscription's client secret
		return c.JSON(fiber.Map{
			"subscription_id": stripeSub.ID,
			"client_secret":   stripeSub.LatestInvoice.PaymentIntent.ClientSecret,
		})
	}

	// Get existing Stripe subscription
	stripeSub, err := subscription.Get(userData.StripeSubscriptionID, nil)
	if err != nil {
		fmt.Printf(
			"Error getting Stripe subscription (%s) for user %s: %+v\n",
			userData.StripeSubscriptionID,
			userData.ID,
			err,
		)
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Return Stripe subscription's client secret
	return c.JSON(fiber.Map{
		"subscription_id": stripeSub.ID,
		"client_secret":   stripeSub.LatestInvoice.PaymentIntent.ClientSecret,
	})
}
