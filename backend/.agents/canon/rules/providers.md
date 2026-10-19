# Providers

- A Provider is an Interface the Core declares  
  and something outside fulfils.
- The Core Depends on the Shape.  
  Never on the library behind it.
- Name the Provider after the business need,  
  never after the vendor.  
  StudentStore, not GormRepository.
- One Struct may Fulfil several Providers.  
  One Provider must never Leak its Vendor.
- The Consumer Declares the Provider it needs,  
  small, and next to the code that calls it.  
  The Composition is the one place that Joins them.
- Comment each Provider with the URL  
  of the contract it wraps.  
  A reader should not have to Search.
- Count the Packages that import a vendor.  
  If the count grows past one, the provider Failed.
- The Word Collides with Angular, NestJS and Terraform,  
  where a provider is a registered dependency.  
  Here it is a Port.
