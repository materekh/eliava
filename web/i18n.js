"use strict";
// UI translations. English strings are the keys; Georgian is the default language.
const KA = {
  // navigation and page titles
  "Home": "მთავარი", "Find a technician": "ხელოსნის ძებნა", "Market": "ბაზარი", "Eliava map": "ელიავას რუკა", "Leaderboard": "რეიტინგი",
  "Post a problem": "პრობლემის გამოქვეყნება", "My requests": "ჩემი მოთხოვნები", "Jobs near me": "სამუშაო ახლოს", "My shop": "ჩემი მაღაზია",
  "My orders": "ჩემი შეკვეთები", "Admin": "ადმინი", "Main menu": "მთავარი მენიუ", "Log in": "შესვლა", "Log out": "გასვლა", "Sign up": "რეგისტრაცია",
  "Create account": "ანგარიშის შექმნა", "My account": "ჩემი ანგარიში", "Account": "ანგარიში", "Parts market": "ნაწილების ბაზარი",
  "Eliava market map": "ელიავას ბაზრობის რუკა", "Eliava: technicians, repairs and parts in Georgia": "ელიავა: ხელოსნები, შეკეთება და ნაწილები საქართველოში",
  "Terms & Conditions": "წესები და პირობები", "Eliava marketplace": "ელიავა", "Map": "რუკა",
  // general
  "Can't reach the server": "სერვერთან დაკავშირება ვერ მოხერხდა",
  "Can't reach the server. It may be waking up, so try again in a minute.": "სერვერთან დაკავშირება ვერ მოხერხდა. შესაძლოა ის ახლა ირთვება, სცადეთ ერთ წუთში.",
  "Request failed ({status})": "მოთხოვნა ვერ შესრულდა ({status})", "Something went wrong": "რაღაც შეცდომა მოხდა",
  "Please log in": "გთხოვთ, შეხვიდეთ სისტემაში", "You need an account for this page.": "ამ გვერდისთვის საჭიროა ანგარიში.",
  "Not available": "მიუწვდომელია", "This page is for {roles} accounts. You are logged in as: {role}.": "ეს გვერდი განკუთვნილია შემდეგი ანგარიშებისთვის: {roles}. თქვენ შესული ხართ როგორც: {role}.",
  "Your registration isn't paid yet, so you are hidden from customers. Pay {fee} for {days} days on the {link} page.": "რეგისტრაცია ჯერ არ არის გადახდილი, ამიტომ მომხმარებლები ვერ გხედავენ. გადაიხადეთ {fee} {days} დღით გვერდზე: {link}.",
  "Photo": "ფოტო", "Search": "ძებნა", "Save": "შენახვა", "Saved": "შენახულია", "Cancel": "გაუქმება", "Refresh": "განახლება", "Name": "სახელი",
  "Nobody yet": "ჯერ არავინ", "Nothing yet.": "ჯერ არაფერია.", "{n} km away": "{n} კმ-ში", "Use my location": "ჩემი მდებარეობა",
  "Your browser can't share location": "თქვენი ბრაუზერი მდებარეობას ვერ აზიარებს",
  "Could not get your location. Click the map instead.": "მდებარეობის დადგენა ვერ მოხერხდა. მონიშნეთ რუკაზე.",
  "Remove {name}": "წაშლა: {name}", "{name}: only photos and videos are allowed": "{name}: დაშვებულია მხოლოდ ფოტოები და ვიდეოები",
  "{name} is too large (max {mb} MB)": "{name} ძალიან დიდია (მაქს. {mb} MB)", "You can upload at most {n} files": "შეგიძლიათ ატვირთოთ მაქსიმუმ {n} ფაილი",
  "Uploading…": "იტვირთება…", "Promoted": "რეკლამა",
  "I'm a technician: join free": "ხელოსანი ვარ: უფასოდ შემოგიერთდებით", "Join as a technician": "შემოუერთდით როგორც ხელოსანი",
  "I'm an Eliava seller: open a shop free": "ელიავაზე ვყიდი: უფასოდ გავხსნი მაღაზიას", "Open a shop": "მაღაზიის გახსნა",
  // reports
  "Report": "დარეპორტება", "Report a problem": "პრობლემის შეტყობინება", "Log in to report": "დასარეპორტებლად შედით სისტემაში",
  "An admin will review it. The person you report is not told who reported them.": "მას ადმინისტრატორი განიხილავს. დარეპორტებულ პირს არ ეცნობება, ვინ დაარეპორტა.",
  "Reason": "მიზეზი", "Details (optional)": "დეტალები (არასავალდებულო)", "Send report": "გაგზავნა",
  "Thanks. An admin will review this report.": "გმადლობთ. ადმინისტრატორი განიხილავს ამ შეტყობინებას.",
  "reason:fake": "ყალბი ანგარიში ან განცხადება", "reason:scam": "თაღლითობა", "reason:unsafe": "საშიში ან უხეში ქცევა", "reason:spam": "სპამი", "reason:other": "სხვა",
  "target:post": "შეკეთების მოთხოვნა", "target:listing": "განცხადება", "target:user": "ანგარიში",
  // home
  "Repairs, technicians and parts, all in one place": "შეკეთება, ხელოსნები და ნაწილები ერთ სივრცეში",
  "Find a nearby technician, post what's broken, or buy parts from the Eliava market with delivery.": "იპოვეთ ხელოსანი ახლოს, გამოაქვეყნეთ რა გაფუჭდა, ან იყიდეთ ნაწილები ელიავას ბაზრობიდან მიტანით.",
  "Search by speciality and distance from you.": "ძებნა სპეციალობით და თქვენგან დაშორებით.", "Search technicians": "ხელოსნების ძებნა",
  "Describe it, add photos/videos and your location. Get a price estimate; nearby technicians contact you.": "აღწერეთ პრობლემა, დაამატეთ ფოტო/ვიდეო და მდებარეობა. მიიღეთ ფასის შეფასება, ახლომდებარე ხელოსნები კი დაგიკავშირდებიან.",
  "Parts and appliances from Eliava market sellers, with stall numbers on the map and home delivery.": "ნაწილები და ტექნიკა ელიავას გამყიდველებისგან, დახლების ნომრებით რუკაზე და სახლამდე მიტანით.",
  "Open market": "ბაზრის გახსნა", "Promoted technicians": "რეკლამირებული ხელოსნები", "Promoted sellers": "რეკლამირებული გამყიდველები",
  "Eliava is just launching": "ელიავა ახლა იწყებს მუშაობას",
  "We're signing up the first technicians and Eliava market sellers in Tbilisi. Join now and be the first customers see.": "ახლა ვარეგისტრირებთ პირველ ხელოსნებს და ელიავას ბაზრობის გამყიდველებს თბილისში. შემოგვიერთდით და იყავით პირველები, ვისაც მომხმარებლები ნახავენ.",
  // technicians
  "Speciality": "სპეციალობა", "All specialities": "ყველა სპეციალობა", "Within (km)": "მანძილი (კმ)",
  "Click the map to set where you need the repair.": "მონიშნეთ რუკაზე, სად გჭირდებათ შეკეთება.", "works within {n} km": "მუშაობს {n} კმ-ის რადიუსში",
  "No technicians found here yet. Eliava is just launching. Post your problem and nearby technicians will see it as they join.": "აქ ჯერ ხელოსნები არ არიან, რადგან ელიავა ახლა იწყებს მუშაობას. გამოაქვეყნეთ პრობლემა და ახლომდებარე ხელოსნები შემოერთებისთანავე ნახავენ.",
  // new post
  "What needs fixing?": "რა გასაკეთებელია?", "Short title": "მოკლე სათაური", "e.g. AC blows warm air": "მაგ. კონდიციონერი თბილ ჰაერს უშვებს",
  "Description (at least 20 characters)": "აღწერა (მინიმუმ 20 სიმბოლო)", "Brand, model, what happens, since when…": "მარკა, მოდელი, რა ხდება, როდიდან…",
  "I already have the needed parts at home": "საჭირო ნაწილები სახლში უკვე მაქვს",
  "Photos and videos (at least one photo; photos up to 10 MB, videos up to 100 MB)": "ფოტოები და ვიდეოები (მინიმუმ ერთი ფოტო; ფოტო 10 MB-მდე, ვიდეო 100 MB-მდე)",
  "Address (optional; technicians only see the area until they view your number)": "მისამართი (არასავალდებულო; ნომრის ნახვამდე ხელოსნები ხედავენ მხოლოდ უბანს)",
  "Location: click the map": "მდებარეობა: მონიშნეთ რუკაზე", "Get price estimate": "ფასის შეფასება", "Post request": "გამოქვეყნება",
  "Estimated cost: {min} – {max}": "სავარაუდო ღირებულება: {min} – {max}", "AI estimate": "AI შეფასება",
  "Pick your location on the map first": "ჯერ მონიშნეთ თქვენი მდებარეობა რუკაზე", "Pick your location on the map": "მონიშნეთ თქვენი მდებარეობა რუკაზე",
  "Add at least one photo of the problem": "დაამატეთ პრობლემის მინიმუმ ერთი ფოტო",
  "Posted, but it looks like a repeat of one of your earlier requests, so it is hidden from technicians until an admin reviews it.": "გამოქვეყნდა, მაგრამ ჰგავს თქვენს ერთ-ერთ წინა მოთხოვნას, ამიტომ ხელოსნებისთვის დამალულია, სანამ ადმინისტრატორი არ განიხილავს.",
  "Your request is live! Estimated {min} – {max}": "მოთხოვნა გამოქვეყნდა! სავარაუდოდ {min} – {max}",
  // posts
  "open": "ღია", "closed": "დახურული", "flagged": "მონიშნული", "has parts": "ნაწილები აქვს", "Estimate {min} – {max}": "შეფასება {min} – {max}",
  "{n} technician(s) viewed your number": "თქვენი ნომერი ნახა {n} ხელოსანმა", "Who viewed?": "ვინ ნახა?", "Mark solved / close": "მოგვარდა / დახურვა",
  "View number ({price})": "ნომრის ნახვა ({price})", "View number (free)": "ნომრის ნახვა (უფასო)",
  "My repair requests": "ჩემი შეკეთების მოთხოვნები", "New request": "ახალი მოთხოვნა",
  "No requests yet. Post your first problem and nearby technicians will see it.": "მოთხოვნები ჯერ არ გაქვთ. გამოაქვეყნეთ პირველი პრობლემა და ახლომდებარე ხელოსნები ნახავენ.",
  "Request closed": "მოთხოვნა დაიხურა",
  // jobs
  "Category": "კატეგორია", "All categories": "ყველა კატეგორია", "Within (km, empty = your service radius)": "მანძილი (კმ; ცარიელი = თქვენი რადიუსი)",
  "Only my specialities": "მხოლოდ ჩემი სპეციალობები", "Set your location and specialities on your Account page first.": "ჯერ ანგარიშის გვერდზე მიუთითეთ მდებარეობა და სპეციალობები.",
  "No open requests nearby right now. Try a bigger distance or untick “Only my specialities”.": "ახლოს ღია მოთხოვნები ჯერ არ არის. გაზარდეთ მანძილი ან მოხსენით „მხოლოდ ჩემი სპეციალობები“.",
  "View this customer's number for {price}? You are charged once per request.": "ნახავთ მომხმარებლის ნომერს {price}-ად? თანხა თითო მოთხოვნაზე ერთხელ ჩამოგეჭრებათ.",
  "View this customer's number? The customer will see that you viewed it.": "ნახავთ მომხმარებლის ნომერს? მომხმარებელი დაინახავს, რომ ნახეთ.",
  // market
  "Eliava stall {code}": "ელიავა, დახლი {code}", "{n} in stock": "მარაგში {n}", "Order": "შეკვეთა", "Quantity": "რაოდენობა",
  "Deliver to my home (+{fee})": "სახლამდე მიტანა (+{fee})",
  "Delivery: Tbilisi only, parcels up to 20 kg. Large appliances: arrange transport with the seller. You pay the seller the item price × quantity (+ delivery fee) in cash on delivery or at pickup.": "მიტანა: მხოლოდ თბილისში, 20 კგ-მდე ამანათები. დიდი ტექნიკის ტრანსპორტირება შეუთანხმეთ გამყიდველს. გამყიდველს უხდით ფასი × რაოდენობა (+ მიტანის საფასური) ნაღდით, მიტანისას ან ადგილზე გატანისას.",
  "Delivery address": "მიტანის მისამართი", "Place order: I will pay on delivery / pickup": "შეკვეთა: გადავიხდი მიტანისას / გატანისას",
  "Order placed! The seller will contact you.": "შეკვეთა მიღებულია! გამყიდველი დაგიკავშირდებათ.", "e.g. compressor": "მაგ. კომპრესორი",
  "Eliava stall": "ელიავას დახლი", "Any stall": "ნებისმიერი დახლი", "Nothing found. Try another search or category.": "ვერაფერი მოიძებნა. სცადეთ სხვა ძებნა ან კატეგორია.",
  "No parts listed yet. Eliava market sellers are joining now.": "ნაწილები ჯერ არ არის დამატებული. ელიავას გამყიდველები ახლა უერთდებიან.",
  "Click a stall to see what its sellers offer.": "დააჭირეთ დახლს, რომ ნახოთ რას გთავაზობენ მისი გამყიდველები.",
  "drag pins to their real positions; changes save automatically.": "გადაიტანეთ ნიშნულები რეალურ ადგილებზე; ცვლილებები ავტომატურად ინახება.",
  "New stall code": "ახალი დახლის კოდი", "Add stall at map centre": "დახლის დამატება რუკის ცენტრში", "Stall {code}": "დახლი {code}",
  "No listings at this stall yet.": "ამ დახლზე განცხადებები ჯერ არ არის.", "Stall {code} moved": "დახლი {code} გადაადგილდა", "Stall added": "დახლი დაემატა",
  // seller
  "Shop details": "მაღაზიის მონაცემები", "Shop name": "მაღაზიის სახელი", "Not at Eliava (pin on map below)": "ელიავაზე არ ვარ (მონიშნეთ რუკაზე)",
  "Address": "მისამართი", "Not at Eliava? Click the map to pin your shop.": "ელიავაზე არ ხართ? მონიშნეთ მაღაზია რუკაზე.", "Save shop": "მაღაზიის შენახვა",
  "New listing": "ახალი განცხადება", "Title": "სათაური", "Price (₾)": "ფასი (₾)", "In stock": "მარაგში", "Description": "აღწერა", "Photos": "ფოტოები",
  "Publish listing": "გამოქვეყნება", "My listings": "ჩემი განცხადებები", "Item": "ნივთი", "Stock": "მარაგი", "Visible": "ჩანს",
  "No listings yet. Add your first one above.": "განცხადებები ჯერ არ გაქვთ. დაამატეთ პირველი ზემოთ.", "Incoming orders": "შემოსული შეკვეთები",
  "Shop saved": "მაღაზია შენახულია", "Listing published": "განცხადება გამოქვეყნდა", "Enter a valid price": "შეიყვანეთ სწორი ფასი",
  // orders
  "No orders yet.": "შეკვეთები ჯერ არ არის.", "Customer": "მომხმარებელი", "Seller": "გამყიდველი", "Qty": "რაოდ.", "Total": "ჯამი",
  "Delivery": "მიტანა", "Status": "სტატუსი", "Pickup": "ადგილზე გატანა", "Order updated": "შეკვეთა განახლდა",
  "status:pending": "მოლოდინში", "status:confirmed": "დადასტურებული", "status:out_for_delivery": "გზაშია", "status:completed": "დასრულებული", "status:cancelled": "გაუქმებული",
  // account
  "Account type": "ანგარიშის ტიპი", "Payments": "გადახდები", "Wallet: {amount}": "საფულე: {amount}", "Top up (₾)": "შევსება (₾)", "Top up": "შევსება",
  "Test mode: no real money is charged.": "სატესტო რეჟიმი: რეალური თანხა არ ჩამოიჭრება.",
  "Free beta: registration and viewing numbers are free. No payments are taken.": "უფასო ბეტა: რეგისტრაცია და ნომრის ნახვა უფასოა. გადახდები არ ხდება.",
  "Registration": "რეგისტრაცია", "Active": "აქტიური", "free during the beta": "უფასოა ბეტა პერიოდში", "until {date}": "{date}-მდე",
  "Not paid": "გადაუხდელი", "you're hidden from customers": "მომხმარებლები ვერ გხედავენ", "Pay {fee} for {days} days": "გადაიხადეთ {fee} {days} დღით",
  "Leaderboard boost": "ტოპში ასვლა",
  "Your active boost: {mine}. Current #1 has {top}. Pay more than that to get to the top for {days} days. Boosted profiles are labelled “Promoted”.": "თქვენი აქტიური ტოპი: {mine}. პირველ ადგილზე მყოფს აქვს {top}. გადაიხადეთ მეტი და იქნებით პირველი {days} დღით. ასეთი პროფილები მონიშნულია წარწერით „რეკლამა“.",
  "Boost amount (₾)": "თანხა (₾)", "Boost": "ტოპი", "Technician profile": "ხელოსნის პროფილი", "City": "ქალაქი", "I travel up to (km)": "მივდივარ მაქსიმუმ (კმ)",
  "About me": "ჩემ შესახებ", "Specialities": "სპეციალობები", "My base location: click the map": "ჩემი ბაზის მდებარეობა: მონიშნეთ რუკაზე",
  "Save profile": "პროფილის შენახვა", "Wallet history": "საფულის ისტორია", "When": "როდის", "What": "რა", "Amount": "თანხა",
  "Wallet topped up": "საფულე შეივსო", "Registration paid": "რეგისტრაცია გადახდილია", "Boost active!": "ტოპი ჩაირთო!",
  "Delete my account": "ანგარიშის წაშლა",
  "Removes your name, email, phone, photos, profile and open requests. Orders and payment records are kept without your personal details.": "იშლება თქვენი სახელი, ელფოსტა, ტელეფონი, ფოტოები, პროფილი და ღია მოთხოვნები. შეკვეთების და გადახდების ჩანაწერები რჩება თქვენი პერსონალური მონაცემების გარეშე.",
  "Password": "პაროლი", "Delete your account permanently? This can't be undone.": "წავშალოთ ანგარიში სამუდამოდ? ამის გაუქმება შეუძლებელია.",
  "Your account was deleted": "ანგარიში წაიშალა", "Set your base location on the map": "მონიშნეთ ბაზის მდებარეობა რუკაზე", "Profile saved": "პროფილი შენახულია",
  // leaderboard
  "Technicians": "ხელოსნები", "Sellers": "გამყიდველები",
  "Promoted placement opens when online payments launch. During the free beta nobody can pay to be first.": "ფასიანი ტოპი ჩაირთვება ონლაინ გადახდების ამოქმედების შემდეგ. უფასო ბეტა პერიოდში პირველ ადგილზე ფულით ვერავინ ავა.",
  "Ranked by boost paid in the last {days} days. Boosted profiles appear first in search and are labelled “Promoted”. Want to be #1? Pay more than the current top on your {link} page.": "რიგი დგინდება ბოლო {days} დღეში ტოპისთვის გადახდილი თანხით. ასეთი პროფილები ძებნაში პირველი ჩანს და მონიშნულია წარწერით „რეკლამა“. გინდათ იყოთ პირველი? გადაიხადეთ მეტი გვერდზე: {link}.",
  "Nobody is promoted yet.": "ჯერ არავინაა ტოპში.",
  // login / register
  "Email": "ელფოსტა", "No account?": "არ გაქვთ ანგარიში?", "I am a…": "მე ვარ…",
  "Customer: I need something fixed or want to buy parts": "მომხმარებელი: რაღაც მაქვს გასაკეთებელი ან ნაწილები მჭირდება",
  "Technician: I fix things ({fee})": "ხელოსანი: ვაკეთებ ტექნიკას ({fee})", "Seller: I sell parts/appliances ({fee})": "გამყიდველი: ვყიდი ნაწილებს/ტექნიკას ({fee})",
  "paid registration": "ფასიანი რეგისტრაცია", "Full name": "სახელი და გვარი", "Phone": "ტელეფონი", "Georgian number, e.g. 555 12 34 56": "ქართული ნომერი, მაგ. 555 12 34 56",
  "Password (8+ characters)": "პაროლი (მინიმუმ 8 სიმბოლო)", "I accept the {link}": "ვეთანხმები: {link}", "Welcome to Eliava!": "კეთილი იყოს თქვენი მობრძანება ელიავაზე!",
  "Enter a Georgian phone number, e.g. +995 555 12 34 56": "შეიყვანეთ ქართული ნომერი, მაგ. +995 555 12 34 56",
  // admin
  "Edit Eliava stall pins": "ელიავას დახლების რედაქტირება", "Reports from users": "მომხმარებლების შეტყობინებები", "Reported by": "ვინ შეატყობინა",
  "{n} reports": "{n} შეტყობინება", "Mark resolved": "მოგვარებულია", "No open reports.": "ღია შეტყობინებები არ არის.", "Report resolved": "შეტყობინება დაიხურა",
  "Who viewed which customer number": "ვინ რომელი მომხმარებლის ნომერი ნახა", "Technician": "ხელოსანი", "Request": "მოთხოვნა", "Row {r}, stall {n}": "რიგი {r}, დახლი {n}", "Paid": "გადახდილი",
  "No views yet.": "ნახვები ჯერ არ არის.", "Flagged posts (possible duplicates)": "მონიშნული მოთხოვნები (შესაძლო გამეორებები)", "Nothing flagged.": "მონიშნული არაფერია.",
  // server error messages
  "A stall with this code already exists": "ამ კოდით დახლი უკვე არსებობს", "Admin accounts can't be deleted here": "ადმინის ანგარიშის აქ წაშლა შეუძლებელია",
  "An account with this email already exists": "ამ ელფოსტით ანგარიში უკვე არსებობს", "Boost between 1 and 10000 GEL": "თანხა უნდა იყოს 1-დან 10 000 ლარამდე",
  "Choose a valid market category": "აირჩიეთ ბაზრის სწორი კატეგორია", "Choose a valid repair category": "აირჩიეთ შეკეთების სწორი კატეგორია",
  "Choose an account type: customer, technician or seller": "აირჩიეთ ანგარიშის ტიპი: მომხმარებელი, ხელოსანი ან გამყიდველი",
  "Choose between 1 and 20 specialities": "აირჩიეთ 1-დან 20-მდე სპეციალობა", "Choose a reason for the report": "აირჩიეთ შეტყობინების მიზეზი",
  "Choose your Eliava stall or pin your shop on the map": "აირჩიეთ დახლი ელიავაზე ან მონიშნეთ მაღაზია რუკაზე", "Code is required": "კოდი სავალდებულოა",
  "Could not read the image": "სურათის წაკითხვა ვერ მოხერხდა", "Enter a valid email address": "შეიყვანეთ სწორი ელფოსტა",
  "Enter a valid phone number": "შეიყვანეთ სწორი ტელეფონის ნომერი", "Enter a valid price in GEL": "შეიყვანეთ სწორი ფასი ლარში",
  "Enter how many you have in stock": "მიუთითეთ, რამდენი გაქვთ მარაგში", "Enter your delivery address": "შეიყვანეთ მიტანის მისამართი",
  "File is too large": "ფაილი ძალიან დიდია", "Image resolution is too large": "სურათის გარჩევადობა ძალიან დიდია", "Invalid form data": "ფორმის მონაცემები არასწორია",
  "Invalid location": "არასწორი მდებარეობა", "Invalid price": "არასწორი ფასი", "Invalid quantity": "არასწორი რაოდენობა", "Invalid stock": "არასწორი მარაგი",
  "Location is required": "მდებარეობა სავალდებულოა", "Not enough balance. Top up your wallet first.": "ბალანსი არ არის საკმარისი. ჯერ შეავსეთ საფულე.",
  "Not found": "ვერ მოიძებნა", "Online payments are not connected yet": "ონლაინ გადახდები ჯერ არ არის ჩართული", "Password must be 8-72 characters": "პაროლი უნდა იყოს 8-დან 72 სიმბოლომდე",
  "Pay the registration fee before boosting": "ტოპისთვის ჯერ გადაიხადეთ რეგისტრაციის საფასური",
  "Pay the seller registration fee to post listings": "განცხადებების დასადებად გადაიხადეთ გამყიდველის რეგისტრაციის საფასური",
  "Pay the technician registration fee to view customer numbers": "ნომრების სანახავად გადაიხადეთ ხელოსნის რეგისტრაციის საფასური",
  "Phone number may only contain digits, spaces, dashes and a leading +": "ნომერი შეიძლება შეიცავდეს მხოლოდ ციფრებს, ჰარებს, ტირეებს და + ნიშანს დასაწყისში",
  "Promoted placement opens when online payments launch": "ფასიანი ტოპი ჩაირთვება ონლაინ გადახდების ამოქმედების შემდეგ",
  "Radius must be between 1 and 500 km": "რადიუსი უნდა იყოს 1-დან 500 კმ-მდე", "Service radius must be between 1 and 500 km": "მომსახურების რადიუსი უნდა იყოს 1-დან 500 კმ-მდე",
  "Set your location in your technician profile first": "ჯერ მიუთითეთ მდებარეობა ხელოსნის პროფილში", "This item is no longer available": "ეს ნივთი აღარ არის ხელმისაწვდომი",
  "This request is closed": "ეს მოთხოვნა დახურულია", "Too many attempts. Please wait and try again later.": "ძალიან ბევრი მცდელობა. მოიცადეთ და სცადეთ მოგვიანებით.",
  "Top up between 1 and 1000 GEL": "შევსება შესაძლებელია 1-დან 1000 ლარამდე", "Unknown speciality": "უცნობი სპეციალობა", "Unknown stall": "უცნობი დახლი",
  "Upload is too large": "ატვირთვა ძალიან დიდია", "Wrong email or password": "ელფოსტა ან პაროლი არასწორია", "Wrong password": "პაროლი არასწორია",
  "You can only cancel an order before the seller confirms it": "შეკვეთის გაუქმება შეიძლება მხოლოდ გამყიდველის დადასტურებამდე",
  "You can't order your own item": "საკუთარი ნივთის შეკვეთა შეუძლებელია", "You can't report yourself": "საკუთარი თავის დარეპორტება შეუძლებელია",
  "You must accept the Terms & Conditions": "უნდა დაეთანხმოთ წესებს და პირობებს", "Your session expired, please log in again": "სესია ამოიწურა, გთხოვთ, ხელახლა შეხვიდეთ",
  "role must be technician or seller": "როლი უნდა იყოს ხელოსანი ან გამყიდველი", "Details": "დეტალები", "Code": "კოდი", "Bio": "ჩემ შესახებ",
  // roles, categories, estimate, wallet
  "role:customer": "მომხმარებელი", "role:technician": "ხელოსანი", "role:seller": "გამყიდველი", "role:admin": "ადმინი",
  "cat:air-conditioner": "კონდიციონერი", "cat:refrigerator": "მაცივარი", "cat:washing-machine": "სარეცხი მანქანა", "cat:dishwasher": "ჭურჭლის სარეცხი მანქანა",
  "cat:tv-electronics": "ტელევიზორი და ელექტრონიკა", "cat:small-appliances": "წვრილი საყოფაცხოვრებო ტექნიკა", "cat:phone-computer": "ტელეფონი და კომპიუტერი",
  "cat:electrical": "ელექტროგაყვანილობა", "cat:plumbing": "სანტექნიკა", "cat:large-gadgets": "მსხვილი ტექნიკა (მაცივარი, სარეცხი, კონდიციონერი)",
  "cat:small-gadgets": "წვრილი ტექნიკა (მიქსერი, ჩაიდანი, უთო)", "cat:ac-parts": "კონდიციონერის ნაწილები", "cat:fridge-parts": "მაცივრის და საყინულის ნაწილები",
  "cat:washer-parts": "სარეცხი მანქანის ნაწილები", "cat:electronic-components": "ელექტრონული კომპონენტები", "cat:tools": "ხელსაწყოები",
  "area:Tbilisi": "თბილისი", "area:Batumi": "ბათუმი", "area:Kutaisi": "ქუთაისი", "area:Rustavi": "რუსთავი", "area:regions": "რეგიონები",
  "customer already has the parts": "ნაწილები უკვე გაქვთ", "includes typical parts": "ტიპური ნაწილების ჩათვლით",
  "Typical price: {cat}, {area}; {parts}.": "ტიპური ფასი: {cat}, {area}; {parts}.",
  "View customer number": "მომხმარებლის ნომრის ნახვა", "Top up (test payment)": "შევსება (სატესტო გადახდა)",
  "Registration fee ({n} days)": "რეგისტრაციის საფასური ({n} დღე)", "Leaderboard boost ({n} days)": "ტოპი ({n} დღე)",
  "{f} is required": "{f}: სავალდებულოა", "{f} must be at least {n} characters": "{f}: მინიმუმ {n} სიმბოლო", "{f} must not exceed {n} characters": "{f}: მაქსიმუმ {n} სიმბოლო",
  "Only {n} left in stock": "მარაგში მხოლოდ {n} დარჩა", "This action is only for {roles} accounts": "ეს მოქმედება მხოლოდ შემდეგი ანგარიშებისთვისაა: {roles}",
  "Can't change order from {a} to {b}": "შეკვეთის სტატუსის შეცვლა შეუძლებელია: {a} → {b}",
  "{name}: only JPG, PNG, WEBP, GIF images and MP4, WEBM, MOV videos are allowed": "{name}: დაშვებულია მხოლოდ JPG, PNG, WEBP, GIF სურათები და MP4, WEBM, MOV ვიდეოები",
  "Invalid {x}": "არასწორი: {x}", "Invalid request": "არასწორი მოთხოვნა",
  // UI rework
  "Account menu": "ანგარიშის მენიუ", "Request": "მოთხოვნა", "Row {r}, stall {n}": "რიგი {r}, დახლი {n}",
  "Jobs": "სამუშაოები",
  "Footer": "ქვედა მენიუ",
  "Free beta: no fees for anyone": "უფასო ბეტა: არანაირი საფასური არავისთვის",
  "How it works": "როგორ მუშაობს",
  "Describe the problem": "აღწერეთ პრობლემა",
  "Add photos, the address area and whether you already have the part. You get a price range right away.": "დაამატეთ ფოტოები, მიუთითეთ უბანი და გაქვთ თუ არა ნაწილი. ფასის მიახლოებით შეფასებას მაშინვე მიიღებთ.",
  "Nearby technicians see it": "ახლომდებარე ხელოსნები ხედავენ მოთხოვნას",
  "Technicians who fix this near you open your request and call you. You see who viewed your number.": "თქვენს ახლოს მყოფი შესაბამისი ხელოსნები ნახულობენ მოთხოვნას და გირეკავენ. თქვენ ხედავთ, ვინ ნახა თქვენი ნომერი.",
  "Get the part from Eliava": "ნაწილი ელიავადან",
  "If a part is needed, order it from a market stall with home delivery or pickup.": "თუ ნაწილია საჭირო, შეუკვეთეთ ბაზრობის დახლიდან სახლამდე მიტანით ან თავად წაიღეთ.",
  "Why it's safe": "რატომ არის უსაფრთხო",
  "Your exact address stays hidden until a technician views your number.": "თქვენი ზუსტი მისამართი დამალულია, სანამ ხელოსანი თქვენს ნომერს არ ნახავს.",
  "You can see every technician who viewed your number.": "ხედავთ ყველა ხელოსანს, ვინც თქვენი ნომერი ნახა.",
  "Every post, listing and profile has a report button, and an admin checks each report.": "ყველა მოთხოვნას, განცხადებას და პროფილს აქვს დარეპორტების ღილაკი. ყოველ რეპორტს ადმინი ამოწმებს.",
  "Location data (GPS) is removed from photos when you upload them.": "ფოტოს ატვირთვისას მისგან მდებარეობის მონაცემები (GPS) იშლება.",
  "Are you a technician or an Eliava seller?": "ხელოსანი ხართ ან ელიავაზე ვაჭრობთ?",
  "Get customers near you. Registration is free during the beta.": "იპოვეთ კლიენტები თქვენს ახლოს. ბეტა პერიოდში რეგისტრაცია უფასოა.",
  "Get customers near you.": "იპოვეთ კლიენტები თქვენს ახლოს.",
  "Click the map or use your location": "მონიშნეთ რუკაზე ან გამოიყენეთ თქვენი მდებარეობა",
  "Location set": "მდებარეობა მონიშნულია",
  "Free. Takes about 2 minutes. Nearby technicians see your request and call you.": "უფასოა და დაახლოებით 2 წუთი სჭირდება. ახლომდებარე ხელოსნები ნახავენ მოთხოვნას და დაგირეკავენ.",
  "What's broken?": "რა გაფუჭდა?",
  "Photos and videos": "ფოტოები და ვიდეოები",
  "Where?": "სად?",
  "Add photos or videos": "ფოტოების ან ვიდეოების დამატება",
  "At least one photo. Photos up to 10 MB, videos up to 100 MB.": "მინიმუმ ერთი ფოტო. ფოტო 10 MB-მდე, ვიდეო 100 MB-მდე.",
  "At least 20 characters: brand, model, what happens and since when.": "მინიმუმ 20 სიმბოლო: მარკა, მოდელი, რა ხდება და როდიდან.",
  "Address (optional)": "მისამართი (არასავალდებულო)",
  "Technicians only see the area until they view your number.": "ნომრის ნახვამდე ხელოსნები ხედავენ მხოლოდ უბანს.",
  "Your service radius": "თქვენი მომსახურების რადიუსი",
  "Pick a stall on the map or from the list to see what its sellers offer.": "აირჩიეთ დახლი რუკაზე ან სიიდან და ნახეთ, რას გთავაზობენ მისი გამყიდველები.",
  "Stalls": "დახლები",
  "No stalls yet.": "დახლები ჯერ არ არის.",
  "Stall positions on the map are approximate until each seller confirms theirs.": "დახლების მდებარეობა რუკაზე მიახლოებითია, სანამ თითოეული გამყიდველი თავისას არ დაადასტურებს.",
  "Add photos": "ფოტოების დამატება",
  "Photos up to 10 MB. The first photo is the cover.": "ფოტო 10 MB-მდე. პირველი ფოტო იქნება მთავარი.",
  "Page not found": "გვერდი ვერ მოიძებნა",
  "This page doesn't exist or was moved.": "გვერდი, რომელსაც ეძებთ, არ არსებობს ან გადატანილია.",
  "Go to the home page": "მთავარ გვერდზე დაბრუნება",
  "Nobody is promoted during the free beta.": "უფასო ბეტა პერიოდში რეკლამირებული პროფილები არ არის.",
};

let LANG = localStorage.getItem("eliava_lang") === "en" ? "en" : "ka";
function setLang(l) { LANG = l === "en" ? "en" : "ka"; localStorage.setItem("eliava_lang", LANG); }
function t(key, vars) {
  let s = (LANG === "ka" && KA[key]) || key;
  if (vars) s = s.replace(/\{(\w+)\}/g, (m, k) => (k in vars ? vars[k] : m));
  return s;
}
function roleName(r) { return LANG === "ka" ? (KA["role:" + r] || r) : r; }
function catName(c) { return (c && LANG === "ka" && KA["cat:" + c.slug]) || c?.name || ""; }
const ERR_PATTERNS = [
  [/^(.+) is required$/, m => t("{f} is required", { f: t(m[1]) })],
  [/^(.+) must be at least (\d+) characters$/, m => t("{f} must be at least {n} characters", { f: t(m[1]), n: m[2] })],
  [/^(.+) must not exceed (\d+) characters$/, m => t("{f} must not exceed {n} characters", { f: t(m[1]), n: m[2] })],
  [/^Only (\d+) left in stock$/, m => t("Only {n} left in stock", { n: m[1] })],
  [/^This action is only for (.+) accounts$/, m => t("This action is only for {roles} accounts", { roles: m[1].split(" / ").map(roleName).join(" / ") })],
  [/^Can't change order from (\S+) to (\S+)$/, m => t("Can't change order from {a} to {b}", { a: t("status:" + m[1]), b: t("status:" + m[2]) })],
  [/^You can upload at most (\d+) files$/, m => t("You can upload at most {n} files", { n: m[1] })],
  [/^(.+) is too large \(max (\d+) MB for \w+\)$/, m => t("{name} is too large (max {mb} MB)", { name: m[1], mb: m[2] })],
  [/^(.+): only JPG, PNG, WEBP, GIF images and MP4, WEBM, MOV videos are allowed$/, m => t("{name}: only JPG, PNG, WEBP, GIF images and MP4, WEBM, MOV videos are allowed", { name: m[1] })],
  [/^Invalid JSON request/, () => t("Invalid request")],
  [/^Invalid (.+)$/, m => t("Invalid {x}", { x: t(m[1]) })],
];
// tErr translates an error message from the API.
function tErr(msg) {
  if (!msg || LANG !== "ka") return msg || "";
  if (KA[msg]) return KA[msg];
  for (const [re, fn] of ERR_PATTERNS) { const m = msg.match(re); if (m) return fn(m); }
  return msg;
}
// detailText translates the leaderboard detail: a shop name with its stall, or a list of category names.
function detailText(d) {
  if (LANG !== "ka" || !d) return d || "";
  return d.replace(" · stall ", " · დახლი ").split(", ").map(n => catName(S.cats.find(c => c.name === n)) || n).join(", ");
}
function estimateNote(e, cat) {
  const m = e.source === "rules" && LANG === "ka" && e.note.match(/^Typical .+ repair in (\w+), (.+)\.$/);
  if (!m) return e.note;
  return t("Typical price: {cat}, {area}; {parts}.", { cat: catName(cat), area: t("area:" + m[1]), parts: t(m[2]) });
}
function reasonText(r) {
  if (LANG !== "ka") return r;
  const m = r.match(/^(Registration fee|Leaderboard boost) \((\d+) days\)$/);
  return m ? t(m[1] + " ({n} days)", { n: m[2] }) : t(r);
}
