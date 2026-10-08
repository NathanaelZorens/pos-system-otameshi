Simple POS System


> Universal View?
+ can
- login
- logout
- register
- view/edit account

> Waiter View
+ can 
- add new order 
- input menu
- checkout
    - cash
    - Cashless
        - QR
        - Card
        - Wallet App?
- put customer name
- print receipt/invoice
- cancel order (before payment)

- toggle sold out for a menu 

> Customer View
+ can
- add new order
    - select in store or out store?
- input menu
- checkout
    - cashless
        - QR
        - Wallet App?
        - Card (only if in store)
    - cash (only if in store)
- view receipt/invoice (not print)
- cancel order (before payment)


> Admin / Owner View
+ can
- view transaction histories
- view statistics?
    - item most/least purchased
    - total money in a day/span of time?
- edit menu / prices + toggle sold out


# Thoughts:
- should customer login? b'cos there are cases of first timer and customer that prolly won't come back (travelling and such), maybe should make cust account "Guest" and "Regular", in which Regular tier has its own perks (point, bonus, discounts, and such)
- Prolly WEB is best, customer can use QR to order. Mobile seems too complicated since need to install... but considering the device is most of the time would be mobile (phones/tablets) Mobile sounds good on paper, cost wise not sure...
- since using web, need to make something that could load fast. make loading time as less as possible (i feel like FE component wise we can do something here, and ofc the Querying)
- since Web, might not need 'hover' actions for buttons and such, but some 'feedback' actions (e.g. on press)
- system will log transactions (UI wise) with data such as: item list, each item prices, total price, paid with what (payment method), Waiter name, Customer Name for that order
- don't know if inventory system would work here (has count of remaining stock), since dish is made with ingredients that is most of the time cannot be accounted for real time (how many vegetables or meat), hence the sold out is toggled, just to make sure customer cannot put order for that. not for "margin vs remaining stock vs profit" kindof thing
- point, bonus, discounts, and such seems to be a whole another feature so this is for later updates. 

